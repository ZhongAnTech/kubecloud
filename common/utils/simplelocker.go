package utils

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	LockerRecordAnnotationKey = "zcloud.kubernetes.io/locker"
	LockerStatusUnkown        = "unkown"
	LockerStatusAcquire       = "acquire"
	LockerStatusRelease       = "release"
	LockerAutoReleaseDuration = int64(60 * time.Second) //90s
)

type LockerRecord struct {
	TimeStamp int64 //nano level
	Identity  string
	Status    string
}

type EndpointLocker struct {
	Namespace string
	Name      string
	Client    corev1.CoreV1Interface
	e         *v1.Endpoints
}

func NewSimpleLocker(namespace, name string, client kubernetes.Interface) *EndpointLocker {
	return &EndpointLocker{
		Namespace: namespace,
		Name:      name,
		Client:    client.CoreV1(),
	}
}

// Get returns the election record from a Endpoints Annotation
func (cl *EndpointLocker) Get() (*LockerRecord, error) {
	var record LockerRecord
	var err error
	cl.e, err = cl.Client.Endpoints(cl.Namespace).Get(cl.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if cl.e.Annotations == nil {
		cl.e.Annotations = make(map[string]string)
	}
	if recordBytes, found := cl.e.Annotations[LockerRecordAnnotationKey]; found {
		if err := json.Unmarshal([]byte(recordBytes), &record); err != nil {
			return nil, err
		}
	}
	return &record, nil
}

// Create attempts to create a LeaderElectionRecord annotation
func (cl *EndpointLocker) Create(ler LockerRecord) error {
	var err error
	cl.e, err = cl.Client.Endpoints(cl.Namespace).Get(cl.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return err
	}
	cl.e, err = cl.Client.Endpoints(cl.Namespace).Create(&v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cl.Name,
			Namespace: cl.Namespace,
			Annotations: map[string]string{
				LockerRecordAnnotationKey: string(recordBytes),
			},
		},
	})

	return err
}

// Update will update and existing annotation on a given resource.
func (cl *EndpointLocker) Update(ler LockerRecord) (string, error) {
	var err error
	if cl.e == nil {
		return LockerStatusUnkown, fmt.Errorf("endpoints not initialized, call get or create first")
	}
	cl.e, err = cl.Client.Endpoints(cl.Namespace).Get(cl.Name, metav1.GetOptions{})
	recordBytes, err := json.Marshal(ler)
	if err != nil {
		return LockerStatusUnkown, err
	}
	info := cl.e.Annotations[LockerRecordAnnotationKey]
	if info != "" && ler.Status == LockerStatusAcquire {
		curLockerRecord := LockerRecord{
			Status: LockerStatusRelease,
		}
		json.Unmarshal([]byte(info), &curLockerRecord)
		now := time.Now().UnixNano()
		timeDiff := int64(0)
		if now >= curLockerRecord.TimeStamp {
			timeDiff = now - curLockerRecord.TimeStamp
		}
		if curLockerRecord.Status == LockerStatusAcquire && timeDiff < LockerAutoReleaseDuration {
			return curLockerRecord.Status, fmt.Errorf("the locker is locked, please wait!")
		}
	}
	cl.e.Annotations[LockerRecordAnnotationKey] = string(recordBytes)
	cl.e, err = cl.Client.Endpoints(cl.Namespace).Update(cl.e)
	return ler.Status, err
}
