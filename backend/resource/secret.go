package resource

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"kubecloud/backend/dao"
	"kubecloud/backend/models"
	"kubecloud/backend/service"
	"kubecloud/common"
	"kubecloud/common/utils"
	"kubecloud/common/validate"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	apiv1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretRes struct {
	cluster    string
	model      *dao.SecretModel
	client     kubernetes.Interface
	listNSFunc NamespaceListFunction
}

type SecretRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Type        apiv1.SecretType  `json:"type"`
	Data        map[string]string `json:"data"`
}

type SecretItem struct {
	SecretRequest `json:",inline"`
	Cluster       string `json:"cluster"`
	Namespace     string `json:"namespace"`
	CreateAt      string `json:"create_at"`
	UpdateAt      string `json:"update_at"`
}

func NewSecretRes(cluster string, get NamespaceListFunction) (*SecretRes, error) {
	client, err := service.GetClientset(cluster)
	if err != nil {
		return nil, fmt.Errorf("get client error %v", err)
	}
	return &SecretRes{
		cluster:    cluster,
		client:     client,
		model:      dao.NewSecretModel(),
		listNSFunc: get,
	}, nil
}

func (res *SecretRes) Exists(namespace, name string) bool {
	return res.model.Exists(res.cluster, namespace, name)
}

func (res *SecretRes) GetTypes() []apiv1.SecretType {
	return []apiv1.SecretType{
		apiv1.SecretTypeTLS,
		apiv1.SecretTypeBasicAuth,
		apiv1.SecretTypeOpaque,
	}
}

func (res *SecretRes) Validate(request *SecretRequest) error {
	err := validate.ValidateString(request.Name)
	if err != nil {
		return common.NewBadRequest().SetCode("SecretInvalidName").SetMessage("invalid secret name").SetCause(err)
	}
	err = validate.ValidateDescription(request.Description)
	if err != nil {
		return common.NewBadRequest().SetCode("SecretInvalidDescription").SetMessage("invalid secret description").SetCause(err)
	}
	switch request.Type {
	case apiv1.SecretTypeTLS:
		{
			cert, _ := request.Data[apiv1.TLSCertKey]
			key, _ := request.Data[apiv1.TLSPrivateKeyKey]
			if cert == "" {
				return common.NewBadRequest().SetCode("SecretRequireCert").SetMessage("TLS cert is required")
			}
			if key == "" {
				return common.NewBadRequest().SetCode("SecretRequireKey").SetMessage("TLS key is required")
			}
			_, err := tls.X509KeyPair([]byte(cert), []byte(key))
			if err != nil {
				return common.NewBadRequest().SetCode("SecretInvalidTLS").SetMessage("invalid TLS cert or key").SetCause(err)
			}
		}
	case apiv1.SecretTypeBasicAuth:
		{
			username, _ := request.Data[apiv1.BasicAuthUsernameKey]
			password, _ := request.Data[apiv1.BasicAuthPasswordKey]
			if username == "" {
				return common.NewBadRequest().SetCode("SecretRequireUsername").SetMessage("username is required")
			}
			if password == "" {
				return common.NewBadRequest().SetCode("SecretRequirePassword").SetMessage("password is required")
			}
		}
	case apiv1.SecretTypeOpaque:
		{
			// bypass
		}
	default:
		{
			return common.NewBadRequest().SetCode("SecretInvalidType").SetMessage("invalid secret type")
		}
	}
	return nil
}

func (res *SecretRes) ListSecret(namespace string, filterQuery *utils.FilterQuery) (*utils.QueryResult, error) {
	nslist := []string{}
	if namespace != common.AllNamespace {
		nslist = append(nslist, namespace)
	} else {
		nslist = res.listNSFunc()
	}
	if len(nslist) == 0 {
		return utils.InitQueryResult([]models.K8sSecret{}, filterQuery), nil
	}
	rows, err := res.model.GetSecretList(res.cluster, nslist, filterQuery)
	if err != nil {
		return nil, err
	}
	list, ok := rows.List.([]models.K8sSecret)
	if !ok {
		err := common.NewInternalServerError().SetCause(fmt.Errorf("invalid data type"))
		return nil, err
	}

	secretList := make([]SecretItem, len(list))
	for i := range list {
		src := list[i]
		dest := &secretList[i]
		dest.Cluster = src.Cluster
		dest.Namespace = src.Namespace
		dest.Name = src.Name
		dest.Type = apiv1.SecretType(src.Type)
		dest.Description = src.Description
		dest.CreateAt = src.CreateAt.Format("2006-01-02 15:04:05")
		dest.UpdateAt = src.UpdateAt.Format("2006-01-02 15:04:05")
		data := map[string]string{}
		if err := json.Unmarshal([]byte(src.Data), &data); err != nil {
			beego.Error("unmarshal secret data failed:", src.Name, err)
			continue
		}
		dest.Data = make(map[string]string, len(data))
		for k, v := range data {
			decoded, err := base64.StdEncoding.DecodeString(v)
			if err != nil {
				beego.Error("base64 decode secret crt data failed:", err)
				continue
			}
			dest.Data[k] = string(decoded)
		}
	}
	rows.List = secretList

	return rows, nil
}

func (res *SecretRes) CreateSecret(namespace string, request *SecretRequest) error {
	if res.model.Exists(res.cluster, namespace, request.Name) {
		return common.NewConflict().SetCode("SecretAlreadyExists").SetMessage("secret already exists")
	}
	k8sSecret := newK8sSecret(namespace, request)
	_, err := res.client.CoreV1().Secrets(namespace).Create(k8sSecret)
	if err != nil {
		return common.FromK8sError(err)
	}
	check := func(param interface{}) error {
		_, err := res.model.GetSecret(res.cluster, namespace, request.Name)
		return err
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP, SYNC_TIMEOUT, check)
	}()
	return <-result
}

func (res *SecretRes) UpdateSecret(namespace string, request *SecretRequest) error {
	client, err := service.GetClientset(res.cluster)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}
	curSecret, err := client.CoreV1().Secrets(namespace).Get(request.Name, metav1.GetOptions{})
	if err != nil {
		return common.FromK8sError(err)
	}
	// check fields
	if curSecret.Type != request.Type {
		return common.NewBadRequest().SetCode("SecretAlterTypeNotAllowed").SetMessage("alter secret type is not allowed")
	}
	if curSecret.Name != request.Name {
		return common.NewBadRequest().SetCode("SecretAlterNameNotAllowed").SetMessage("alter secret name is not allowed")
	}
	// update fields
	newSecret := newK8sSecret(namespace, request)
	curSecret.Annotations = newSecret.Annotations
	curSecret.Data = newSecret.Data
	_, err = res.client.CoreV1().Secrets(namespace).Update(curSecret)
	if err != nil {
		return common.NewInternalServerError().SetCause(err)
	}

	return nil
}

func (res *SecretRes) DeleteSecret(namespace, name string) error {
	err := res.client.CoreV1().Secrets(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			err = res.model.DeleteSecret(res.cluster, namespace, name)
			if err == orm.ErrNoRows {
				return nil
			}
		}
		return common.FromK8sError(err)
	}
	if err != nil {
		return err
	}
	check := func(param interface{}) error {
		_, err := res.model.GetSecret(res.cluster, namespace, name)
		if err == orm.ErrNoRows {
			return nil
		}
		return err
	}
	result := make(chan error)
	go func() {
		result <- WaitSync(nil, SYNC_CHECK_STEP, SYNC_TIMEOUT, check)
	}()
	return <-result
}

func newK8sSecret(namespace string, request *SecretRequest) *apiv1.Secret {
	data := make(map[string][]byte)
	for k, v := range request.Data {
		data[k] = []byte(v)
	}

	annotations := make(map[string]string)
	annotations[DescriptionAnnotationKey] = request.Description

	return &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        request.Name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Data: data,
		Type: request.Type,
	}
}
