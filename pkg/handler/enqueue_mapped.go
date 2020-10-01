/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

// MapFunc is the signature required for enqueueing requests from a generic function.
// This type is usually used with EnqueueRequestsFromMapFunc when registering an event handler.
type MapFunc func(MapObject) []reconcile.Request

// EnqueueRequestsFromMapFunc enqueues Requests by running a transformation function that outputs a collection
// of reconcile.Requests on each Event.  The reconcile.Requests may be for an arbitrary set of objects
// defined by some user specified transformation of the source Event.  (e.g. trigger Reconciler for a set of objects
// in response to a cluster resize event caused by adding or deleting a Node)
//
// EnqueueRequestsFromMapFunc is frequently used to fan-out updates from one object to one or more other
// objects of a differing type.
//
// For UpdateEvents which contain both a new and old object, the transformation function is run on both
// objects and both sets of Requests are enqueue.
func EnqueueRequestsFromMapFunc(fn MapFunc) EventHandler {
	return &enqueueRequestsFromMapFunc{
		toRequests: fn,
	}
}

var _ EventHandler = &enqueueRequestsFromMapFunc{}

type enqueueRequestsFromMapFunc struct {
	// Mapper transforms the argument into a slice of keys to be reconciled
	toRequests MapFunc
}

// Create implements EventHandler
func (e *enqueueRequestsFromMapFunc) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, MapObject{Object: evt.Object})
}

// Update implements EventHandler
func (e *enqueueRequestsFromMapFunc) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, MapObject{Object: evt.ObjectOld})
	e.mapAndEnqueue(q, MapObject{Object: evt.ObjectNew})
}

// Delete implements EventHandler
func (e *enqueueRequestsFromMapFunc) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, MapObject{Object: evt.Object})
}

// Generic implements EventHandler
func (e *enqueueRequestsFromMapFunc) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.mapAndEnqueue(q, MapObject{Object: evt.Object})
}

func (e *enqueueRequestsFromMapFunc) mapAndEnqueue(q workqueue.RateLimitingInterface, object MapObject) {
	for _, req := range e.toRequests(object) {
		q.Add(req)
	}
}

// EnqueueRequestsFromMapFunc can inject fields into the mapper.

// InjectFunc implements inject.Injector.
func (e *enqueueRequestsFromMapFunc) InjectFunc(f inject.Func) error {
	if f == nil {
		return nil
	}
	return f(e.toRequests)
}

// MapObject contains information from an event to be transformed into a Request.
type MapObject struct {
	Object client.Object
}
