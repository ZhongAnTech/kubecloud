package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"gopkg.in/igm/sockjs-go.v2/sockjs"

	// remotecommandconsts "k8s.io/apimachinery/pkg/util/remotecommand"
	"bytes"
	"path"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"kubecloud/backend/resource"
	"kubecloud/backend/service"
	"kubecloud/common"
)

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// TerminalSession implements PtyHandler (using a SockJS connection)
type TerminalSession struct {
	Id            string `json:"id"`
	bound         chan error
	sockJSSession sockjs.Session
	sizeChan      chan remotecommand.TerminalSize
}

// TerminalMessage is the messaging protocol between ShellController and TerminalSession.
//
// OP      DIRECTION  FIELD(S) USED  DESCRIPTION
// ---------------------------------------------------------------------
// bind    fe->be     SessionID      Id sent back from TerminalReponse
// stdin   fe->be     Data           Keystrokes/paste buffer
// resize  fe->be     Rows, Cols     New terminal size
// stdout  be->fe     Data           Output from the process
// toast   be->fe     Data           OOB message to be shown to the user
type TerminalMessage struct {
	Op, Data, SessionID string
	Rows, Cols          uint16
}

// TerminalSize handles pty->process resize events
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Next() *remotecommand.TerminalSize {
	select {
	case size := <-t.sizeChan:
		return &size
	}
}

// Read handles pty->process messages (stdin, resize)
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Read(p []byte) (int, error) {
	m, err := t.sockJSSession.Recv()
	if err != nil {
		return 0, err
	}

	var msg TerminalMessage
	if err := json.Unmarshal([]byte(m), &msg); err != nil {
		return 0, err
	}

	switch msg.Op {
	case "stdin":
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

// Write handles process->pty stdout
// Called from remotecommand whenever there is any output
func (t TerminalSession) Write(p []byte) (int, error) {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		return 0, err
	}

	if err = t.sockJSSession.Send(string(msg)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Toast can be used to send the user any OOB messages
// hterm puts these in the center of the terminal
func (t TerminalSession) Toast(p string) error {
	msg, err := json.Marshal(TerminalMessage{
		Op:   "toast",
		Data: p,
	})
	if err != nil {
		return err
	}

	if err = t.sockJSSession.Send(string(msg)); err != nil {
		return err
	}
	return nil
}

// Close shuts down the SockJS connection and sends the status code and reason to the client
// Can happen if the process exits or if there is an error starting up the process
// For now the status code is unused and reason is shown to the user (unless "")
func (t TerminalSession) Close(status uint32, reason string) {
	t.sockJSSession.Close(status, reason)
}

// terminalSessions stores a map of all TerminalSession objects
// FIXME: this structure needs locking
var terminalSessions = make(map[string]TerminalSession)

// handleTerminalSession is Called by net/http for any new /api/sockjs connections
func handleTerminalSession(session sockjs.Session) {
	var (
		buf             string
		err             error
		msg             TerminalMessage
		terminalSession TerminalSession
		ok              bool
	)

	if buf, err = session.Recv(); err != nil {
		log.Printf("handleTerminalSession: can't Recv: %v", err)
		return
	}

	if err = json.Unmarshal([]byte(buf), &msg); err != nil {
		log.Printf("handleTerminalSession: can't UnMarshal (%v): %s", err, buf)
		return
	}

	if msg.Op != "bind" {
		log.Printf("handleTerminalSession: expected 'bind' message, got: %s", buf)
		return
	}

	if terminalSession, ok = terminalSessions[msg.SessionID]; !ok {
		log.Printf("handleTerminalSession: can't find session '%s'", msg.SessionID)
		return
	}

	terminalSession.sockJSSession = session
	terminalSession.bound <- nil
	terminalSessions[msg.SessionID] = terminalSession
}

// CreateAttachHandler is called from main for /api/sockjs
func CreateAttachHandler(path string) http.Handler {
	return sockjs.NewHandler(path, sockjs.DefaultOptions, handleTerminalSession)
}

func parseExecOption(request *rest.Request, options ExecOption) *rest.Request {
	request = request.Param("tty", strconv.FormatBool(options.tty)).
		Param("stderr", strconv.FormatBool(options.stderr)).
		Param("stdout", strconv.FormatBool(options.stdout)).
		Param("stdin", strconv.FormatBool(options.stdin)).
		Param("container", options.containerName)

	for _, value := range options.cmd {
		request = request.Param("command", value)
	}
	return request
}

// startProcess is called by handleAttach
// Executed cmd in the container specified in request and connects it up with the ptyHandler (a session)
func startProcess(k8sClient kubernetes.Interface, cfg *rest.Config, namespace, podName, containerName string, cmd []string, ptyHandler PtyHandler) error {
	req := k8sClient.Core().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req = parseExecOption(req, ExecOption{
		stdin:         true,
		stdout:        true,
		stderr:        true,
		tty:           true,
		cmd:           cmd,
		containerName: containerName,
	})

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		// SupportedProtocols: remotecommandconsts.SupportedStreamingProtocols,
		Stdin:             ptyHandler,
		Stdout:            ptyHandler,
		Stderr:            ptyHandler,
		TerminalSizeQueue: ptyHandler,
		Tty:               true,
	})
	if err != nil {
		return err
	}

	return nil
}

// genTerminalSessionId generates a random session ID string. The format is not really interesting.
// This ID is used to identify the session when the client opens the SockJS connection.
// Not the same as the SockJS session id! We can't use that as that is generated
// on the client side and we don't have it yet at this point.
func genTerminalSessionId() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	id := make([]byte, hex.EncodedLen(len(bytes)))
	hex.Encode(id, bytes)
	return string(id), nil
}

// WaitForTerminal is called from apihandler.handleAttach as a goroutine
// Waits for the SockJS connection to be opened by the client the session to be bound in handleTerminalSession
func WaitForTerminal(k8sClient kubernetes.Interface, cfg *rest.Config, namespace, podName, containerName, sessionId string) {

	select {
	case <-terminalSessions[sessionId].bound:
		close(terminalSessions[sessionId].bound)

		session := terminalSessions[sessionId]
		// release session resource
		delete(terminalSessions, sessionId)

		var err error
		validShells := []string{"bash", "sh"}

		// No shell given or it was not valid: try some shells until one succeeds or all fail
		// FIXME: if the first shell fails then the first keyboard event is lost
		for _, testShell := range validShells {
			cmd := []string{testShell}
			if err = startProcess(k8sClient, cfg, namespace, podName, containerName, cmd, session); err == nil {
				break
			}
		}

		session.Toast("Disconnected")
		if err != nil {
			session.Close(2, err.Error())
			beego.Error("Error occurred when connect container with session id:", sessionId)
		} else {
			session.Close(1, "Process exited")
			beego.Debug("Close container connection with session id:", sessionId)
		}
	}
}

type TermController struct {
	BaseController
}

type ExecOption struct {
	stdin         bool
	stdout        bool
	stderr        bool
	tty           bool
	cmd           []string
	containerName string
}

func (this *TermController) PodTerminal() {
	cluster := this.Ctx.Input.Param(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	podName := this.Ctx.Input.Param(":podname")
	containerName := this.Ctx.Input.Param(":containername")

	sessionId, err := genTerminalSessionId()
	if err != nil {
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	k8sClient, err := service.GetClientset(cluster)
	if err != nil {
		this.Data["json"] = NewResult(false, nil, err.Error())
		this.ServeJSON()
		return
	}

	configFile := path.Join(beego.AppConfig.String("k8s::configPath"), cluster)
	cfg, err := clientcmd.BuildConfigFromFlags("", configFile)
	if err != nil {
		this.Data["json"] = NewResult(false, nil, err.Error())
		this.ServeJSON()
		return
	}

	terminalSession := TerminalSession{
		Id:       sessionId,
		bound:    make(chan error),
		sizeChan: make(chan remotecommand.TerminalSize),
	}
	terminalSessions[sessionId] = terminalSession

	beego.Debug("Terminal sessions add a new session with session id:", sessionId)
	beego.Debug("Terminal sessions' length is:", len(terminalSessions))

	go WaitForTerminal(k8sClient, cfg, namespace, podName, containerName, sessionId)

	this.Data["json"] = NewResult(true, terminalSession, "")
	this.ServeJSON()
}

type ExecController struct {
	BaseController
}

func (this *ExecController) PodTerminalExec() {
	cluster := this.Ctx.Input.Param(":cluster")
	namespace := this.Ctx.Input.Param(":namespace")
	podName := this.Ctx.Input.Param(":podname")
	containerName := this.Ctx.Input.Param(":containername")

	execRequestBody := resource.ExecRequest{}
	err := json.Unmarshal([]byte(this.Ctx.Input.RequestBody), &execRequestBody)
	if err != nil {
		beego.Error("Alert request body parse error: ", err.Error())
		this.ServeError(common.NewBadRequest().SetCause(err))
		return
	}

	k8sClient, err := service.GetClientset(cluster)
	if err != nil {
		beego.Error("Get k8s client error: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	configFile := path.Join(beego.AppConfig.String("k8s::configPath"), cluster)
	cfg, err := clientcmd.BuildConfigFromFlags("", configFile)
	if err != nil {
		beego.Error("Build config from file error: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	cmd := strings.Split(trimAndformat(execRequestBody.Command), " ")

	req := k8sClient.Core().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req = parseExecOption(req, ExecOption{
		stdin:         false,
		stdout:        true,
		stderr:        true,
		tty:           false,
		cmd:           cmd,
		containerName: containerName,
	})

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		beego.Error("Get SPDY exec error: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}
	var stdout, stderr bytes.Buffer

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		beego.Error("Exec cmd<", cmd, "> error: ", err.Error())
		this.ServeError(common.NewInternalServerError().SetCause(err))
		return
	}

	this.Data["json"] = NewResult(true, stdout.String(), "")
	this.ServeJSON()
}

func trimAndformat(s string) string {
	sSlice := strings.Split(s, " ")
	result := ""
	for _, item := range sSlice {
		if item != "" {
			result += item + " "
		}
	}
	return strings.TrimSpace(result)
}
