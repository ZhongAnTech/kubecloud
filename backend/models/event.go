package models

import (
	"time"
)

type ZcloudEvent struct {
	ID              int64     `orm:"pk;column(id);auto" json:"id"`
	EventUid        string    `orm:"column(event_uid);size(36)" json:"event_uid"`
	ActionType      string    `orm:"column(action_type);size(10)" json:"action_type"`
	EventType       string    `orm:"column(event_type);size(10)" json:"event_type"`
	Cluster         string    `orm:"column(cluster)" json:"cluster"`
	Namespace       string    `orm:"column(namespace);size(100)" json:"namespace"`
	SourceComponent string    `orm:"column(source_component);size(50)" json:"source_component"`
	SourceHost      string    `orm:"column(source_host);size(20)" json:"source_host"`
	ObjectKind      string    `orm:"column(object_kind);size(20)" json:"object_kind"`
	ObjectName      string    `orm:"column(object_name);size(100)" json:"object_name"`
	ObjectUid       string    `orm:"column(object_uid);size(36)" json:"object_uid"`
	FieldPath       string    `orm:"column(field_path);size(200)" json:"field_path"`
	Reason          string    `orm:"column(reason);size(100)" json:"reason"`
	Message         string    `orm:"column(message);type(text)" json:"message"`
	Count           int32     `orm:"column(count)" json:"count"`
	FirstTimestamp  time.Time `orm:"column(first_time)" json:"first_time"`
	LastTimestamp   time.Time `orm:"column(last_time);index" json:"last_time"`
}

func (t *ZcloudEvent) TableName() string {
	return "zcloud_event"
}
