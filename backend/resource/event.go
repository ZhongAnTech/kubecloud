package resource

import (
	"kubecloud/backend/dao"
	"time"
)

type eventInfo struct {
	ClusterName   string    `json:"cluster_name"`
	Namespace     string    `json:"namespace"`
	EventLevel    string    `json:"event_level"`
	ObjectKind    string    `json:"object_kind"`
	ObjectName    string    `json:"object_name"`
	ObjectPhase   string    `json:"object_phase"`
	ObjectMessage string    `json:"object_message"`
	SourceHost    string    `json:"source_host"`
	LastTime      time.Time `json:"last_time"`
}

func GetEvents(clusterName, namespace, sourceHost, objectKind, objectName, eventLevel string, limitCount int64) ([]eventInfo, error) {
	eventsInfo := []eventInfo{}
	events, err := dao.GetEvents(clusterName, namespace, sourceHost, objectKind, objectName, eventLevel, limitCount)
	if err != nil {
		return eventsInfo, err
	}
	for _, event := range events {
		eventsInfo = append(eventsInfo, eventInfo{
			ClusterName:   event.Cluster,
			Namespace:     event.Namespace,
			EventLevel:    event.EventType,
			ObjectKind:    event.ObjectKind,
			ObjectName:    event.ObjectName,
			ObjectPhase:   event.Reason,
			ObjectMessage: event.Message,
			SourceHost:    event.SourceHost,
			LastTime:      event.LastTimestamp,
		})
	}
	return eventsInfo, nil
}
