// Copyright 2019 Altinity Ltd and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chi

import (
	"context"
	"fmt"
	"time"

	log "github.com/golang/glog"

	a "github.com/altinity/clickhouse-operator/pkg/announcer"
	chiV1 "github.com/altinity/clickhouse-operator/pkg/apis/clickhouse.altinity.com/v1"
)

// Announcer handler all log/event/status messages going outside of controller/worker
type Announcer struct {
	a.Announcer

	ctrl *Controller
	chi  *chiV1.ClickHouseInstallation

	// writeEvent specifies whether to produce k8s event into chi, therefore requires chi to be specified
	// See k8s event for details.
	// https://kubernetes.io/docs/reference/kubernetes-api/cluster-resources/event-v1/
	writeEvent bool
	// eventAction specifies k8s event action
	eventAction string
	// event reason specifies k8s event reason
	eventReason string

	// writeStatusAction specifies whether to produce action into `ClickHouseInstallation.Status.Action` of chi,
	// therefore requires chi to be specified
	writeStatusAction bool
	// writeStatusAction specifies whether to produce action into `ClickHouseInstallation.Status.Actions` of chi,
	// therefore requires chi to be specified
	writeStatusActions bool
	// writeStatusAction specifies whether to produce action into `ClickHouseInstallation.Status.Error` of chi,
	// therefore requires chi to be specified
	writeStatusError bool
}

// NewAnnouncer creates new announcer
func NewAnnouncer() Announcer {
	return Announcer{
		Announcer: a.New(),
	}
}

// Silence produces silent announcer
func (a Announcer) Silence() Announcer {
	b := a
	b.Announcer = b.Announcer.Silence()
	return b
}

// V is inspired by log.V()
func (a Announcer) V(level log.Level) Announcer {
	b := a
	b.Announcer = b.Announcer.V(level)
	return b
}

// F adds function name
func (a Announcer) F() Announcer {
	b := a
	b.Announcer = b.Announcer.F()
	return b
}

// L adds line number
func (a Announcer) L() Announcer {
	b := a
	b.Announcer = b.Announcer.L()
	return b
}

// FL adds filename
func (a Announcer) FL() Announcer {
	b := a
	b.Announcer = b.Announcer.FL()
	return b
}

// A adds full code address as 'file:line:function'
func (a Announcer) A() Announcer {
	b := a
	b.Announcer = b.Announcer.A()
	return b
}

// S adds 'start of the function' tag
func (a Announcer) S() Announcer {
	b := a
	b.Announcer = b.Announcer.S()
	return b
}

// E adds 'end of the function' tag
func (a Announcer) E() Announcer {
	b := a
	b.Announcer = b.Announcer.E()
	return b
}

// M adds object meta as 'namespace/name'
func (a Announcer) M(m ...interface{}) Announcer {
	b := a
	b.Announcer = b.Announcer.M(m...)
	return b
}

// P triggers log to print line
func (a Announcer) P() {
	a.Info("")
}

// Info is inspired by log.Infof()
func (a Announcer) Info(format string, args ...interface{}) {
	// Produce classic log line
	a.Announcer.Info(format, args...)

	// Produce k8s event
	if a.writeEvent && a.chiCapable() {
		if len(args) > 0 {
			a.ctrl.EventInfo(a.chi, a.eventAction, a.eventReason, fmt.Sprintf(format, args...))
		} else {
			a.ctrl.EventInfo(a.chi, a.eventAction, a.eventReason, fmt.Sprint(format))
		}
	}

	// Produce chi status record
	a.writeCHIStatus(format, args...)
}

// Warning is inspired by log.Warningf()
func (a Announcer) Warning(format string, args ...interface{}) {
	// Produce classic log line
	a.Announcer.Warning(format, args...)

	// Produce k8s event
	if a.writeEvent && a.chiCapable() {
		if len(args) > 0 {
			a.ctrl.EventWarning(a.chi, a.eventAction, a.eventReason, fmt.Sprintf(format, args...))
		} else {
			a.ctrl.EventWarning(a.chi, a.eventAction, a.eventReason, fmt.Sprint(format))
		}
	}

	// Produce chi status record
	a.writeCHIStatus(format, args...)
}

// Error is inspired by log.Errorf()
func (a Announcer) Error(format string, args ...interface{}) {
	// Produce classic log line
	a.Announcer.Error(format, args...)

	// Produce k8s event
	if a.writeEvent && a.chiCapable() {
		if len(args) > 0 {
			a.ctrl.EventError(a.chi, a.eventAction, a.eventReason, fmt.Sprintf(format, args...))
		} else {
			a.ctrl.EventError(a.chi, a.eventAction, a.eventReason, fmt.Sprint(format))
		}
	}

	// Produce chi status record
	a.writeCHIStatus(format, args...)
}

// Fatal is inspired by log.Fatalf()
func (a Announcer) Fatal(format string, args ...interface{}) {
	// Produce k8s event
	if a.writeEvent && a.chiCapable() {
		if len(args) > 0 {
			a.ctrl.EventError(a.chi, a.eventAction, a.eventReason, fmt.Sprintf(format, args...))
		} else {
			a.ctrl.EventError(a.chi, a.eventAction, a.eventReason, fmt.Sprint(format))
		}
	}

	// Produce chi status record
	a.writeCHIStatus(format, args...)

	// Write and exit
	a.Announcer.Fatal(format, args...)
}

// WithController specifies controller to be used in case `chi`-related announces need to be done
func (a Announcer) WithController(ctrl *Controller) Announcer {
	b := a
	b.ctrl = ctrl
	return b
}

// WithEvent is used in chained calls in order to produce event into `chi`
func (a Announcer) WithEvent(
	chi *chiV1.ClickHouseInstallation,
	action string,
	reason string,
) Announcer {
	b := a
	if chi == nil {
		b.writeEvent = false
		b.chi = nil
		b.eventAction = ""
		b.eventReason = ""
	} else {
		b.writeEvent = true
		b.chi = chi
		b.eventAction = action
		b.eventReason = reason
	}
	return b
}

// WithStatusAction is used in chained calls in order to produce action into `ClickHouseInstallation.Status.Action`
func (a Announcer) WithStatusAction(chi *chiV1.ClickHouseInstallation) Announcer {
	b := a
	if chi == nil {
		b.chi = nil
		b.writeStatusAction = false
	} else {
		b.chi = chi
		b.writeStatusAction = true
	}
	return b
}

// WithStatusActions is used in chained calls in order to produce action in ClickHouseInstallation.Status.Actions
func (a Announcer) WithStatusActions(chi *chiV1.ClickHouseInstallation) Announcer {
	b := a
	if chi == nil {
		b.chi = nil
		b.writeStatusActions = false
	} else {
		b.chi = chi
		b.writeStatusActions = true
	}
	return b
}

// WithStatusError is used in chained calls in order to produce error in ClickHouseInstallation.Status.Error
func (a Announcer) WithStatusError(chi *chiV1.ClickHouseInstallation) Announcer {
	b := a
	if chi == nil {
		b.chi = nil
		b.writeStatusError = false
	} else {
		b.chi = chi
		b.writeStatusError = true
	}
	return b
}

// chiCapable checks whether announcer is capable to produce chi-based announcements
func (a Announcer) chiCapable() bool {
	return (a.ctrl != nil) && (a.chi != nil)
}

// writeCHIStatus is internal function which writes ClickHouseInstallation.Status
func (a Announcer) writeCHIStatus(format string, args ...interface{}) {
	if !a.chiCapable() {
		return
	}

	now := time.Now()
	prefix := now.Format(time.RFC3339Nano) + " "

	if a.writeStatusAction {
		if len(args) > 0 {
			a.chi.EnsureStatus().SetAction(fmt.Sprintf(format, args...))
		} else {
			a.chi.EnsureStatus().SetAction(fmt.Sprint(format))
		}
	}
	if a.writeStatusActions {
		if len(args) > 0 {
			a.chi.EnsureStatus().PushAction(prefix + fmt.Sprintf(format, args...))
		} else {
			a.chi.EnsureStatus().PushAction(prefix + fmt.Sprint(format))
		}
	}
	if a.writeStatusError {
		if len(args) > 0 {
			// PR review question: should we prefix the string in the SetError call? If so, we can SetAndPushError.
			a.chi.EnsureStatus().SetError(fmt.Sprintf(format, args...))
			a.chi.EnsureStatus().PushError(prefix + fmt.Sprintf(format, args...))
		} else {
			a.chi.EnsureStatus().SetError(fmt.Sprint(format))
			a.chi.EnsureStatus().PushError(prefix + fmt.Sprint(format))
		}
	}

	// Propagate status updates into object
	if a.writeStatusAction || a.writeStatusActions || a.writeStatusError {
		_ = a.ctrl.updateCHIObjectStatus(context.Background(), a.chi, UpdateCHIStatusOptions{
			TolerateAbsence: true,
			CopyCHIStatusOptions: chiV1.CopyCHIStatusOptions{
				Actions: true,
				Errors:  true,
			},
		})
	}
}
