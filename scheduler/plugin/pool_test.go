/*
   Copyright 2023 The Kubernetes Authors.

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

package wasm

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

var ctx = context.Background()

type testGuest struct {
	val int
}

func Test_guestPool_getForScheduling(t *testing.T) {
	uid := uuid.NewUUID()
	differentUID := uuid.NewUUID()

	var counter int
	pl, err := newGuestPool(ctx, func(ctx2 context.Context) (*testGuest, error) {
		counter++
		return &testGuest{val: counter}, nil
	})
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	var g1 *testGuest
	if err = pl.doWithSchedulingGuest(ctx, uid, func(t *testGuest) {
		g1 = t
	}); err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g1 == nil {
		t.Fatalf("have nil guest instance")
	}

	// Scheduling is sequential, so we expect a different ID to re-use the prior
	var g2 *testGuest
	if err = pl.doWithSchedulingGuest(ctx, differentUID, func(t *testGuest) {
		g2 = t
	}); err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}
	if g2 == nil {
		t.Fatalf("have nil guest instance")
	}
	if want, have := g1, g2; !reflect.DeepEqual(want, have) {
		t.Fatalf("expected the same guest: want %v, have %v", want, have)
	}
}

func Test_guestPool_getForBinding(t *testing.T) {
	uid := uuid.NewUUID()
	differentUID := uuid.NewUUID()

	var counter int
	pl, err := newGuestPool(ctx, func(ctx2 context.Context) (*testGuest, error) {
		counter++
		return &testGuest{val: counter}, nil
	})
	if err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// assign for scheduling
	var g1 *testGuest
	if err = pl.doWithSchedulingGuest(ctx, uid, func(t *testGuest) {
		g1 = t
	}); err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// reassign for binding
	pl.getForBinding(uid)

	if pl.scheduledPodUID != "" {
		t.Fatalf("expected no scheduling cycles")
	}

	if pl.scheduled != nil {
		t.Fatalf("expected no scheduling cycles")
	}

	// assign another for scheduling
	var g2 *testGuest
	if err = pl.doWithSchedulingGuest(ctx, differentUID, func(t *testGuest) {
		g2 = t
	}); err != nil {
		t.Fatalf("failed to get guest instance: %v", err)
	}

	// reassign it for binding
	pl.getForBinding(differentUID)

	if want, have := map[types.UID]*testGuest{uid: g1, differentUID: g2}, pl.binding; !reflect.DeepEqual(want, have) {
		t.Fatalf("expected two guests in the binding cycle: want %v, have %v", want, have)
	}
}
