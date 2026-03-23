// Copyright Istio Authors
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

package kstatus

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ---------------------------------------------------------------------------
// Reason types — one distinct Go type per condition family.
// This makes (conditionType, reason) pairings a compile-time check.
// ---------------------------------------------------------------------------

type GatewayAcceptedReason string
type GatewayProgrammedReason string
type ListenerAcceptedReason string
type ListenerProgrammedReason string
type ListenerConflictedReason string
type ListenerResolvedRefsReason string
type RouteAcceptedReason string
type RouteResolvedRefsReason string
type RouteResolvedWaypointsReason string
type PolicyAcceptedReason string
type BackendTLSResolvedRefsReason string
type InferencePoolAcceptedReason string
type InferencePoolResolvedRefsReason string
type GatewayClassAcceptedReason string

// ---------------------------------------------------------------------------
// Internal entry — mirrors the old gateway/conditions.go `condition` struct
// ---------------------------------------------------------------------------

type condEntry struct {
	condType string
	reason   string
	message  string
	// status is the success-polarity status. Empty defaults to ConditionTrue.
	status metav1.ConditionStatus
	// err, when non-nil, replaces reason/message and inverts the polarity.
	err *condEntryError
	// setOnce, when non-empty, uses CreateCondition instead of UpdateConditionIfChanged.
	setOnce string
}

type condEntryError struct {
	reason  string
	message string
}

// applyAll writes all entries into existingConditions and returns the result.
func applyAll(generation int64, existingConditions []metav1.Condition, entries []condEntry) []metav1.Condition {
	for _, e := range entries {
		setter := UpdateConditionIfChanged
		if e.setOnce != "" {
			setOnceReason := e.setOnce
			setter = func(conds []metav1.Condition, cond metav1.Condition) []metav1.Condition {
				return CreateCondition(conds, cond, setOnceReason)
			}
		}
		if e.err != nil {
			successStatus := e.status
			if successStatus == "" {
				successStatus = StatusTrue
			}
			existingConditions = setter(existingConditions, metav1.Condition{
				Type:               e.condType,
				Status:             InvertStatus(successStatus),
				ObservedGeneration: generation,
				LastTransitionTime: metav1.Now(),
				Reason:             e.err.reason,
				Message:            e.err.message,
			})
		} else {
			status := e.status
			if status == "" {
				status = StatusTrue
			}
			existingConditions = setter(existingConditions, metav1.Condition{
				Type:               e.condType,
				Status:             status,
				ObservedGeneration: generation,
				LastTransitionTime: metav1.Now(),
				Reason:             e.reason,
				Message:            e.message,
			})
		}
	}
	return existingConditions
}

// ---------------------------------------------------------------------------
// GatewayConditionSet — Accepted + Programmed
// ---------------------------------------------------------------------------

type GatewayConditionSet struct {
	accepted   condEntry
	programmed condEntry
}

const (
	gatewayCondAccepted   = "Accepted"
	gatewayCondProgrammed = "Programmed"
)

// NewGatewayConditionSet creates a set pre-populated with the default success state.
func NewGatewayConditionSet(acceptedReason GatewayAcceptedReason, acceptedMsg string,
	programmedReason GatewayProgrammedReason, programmedMsg string,
) GatewayConditionSet {
	return GatewayConditionSet{
		accepted: condEntry{
			condType: gatewayCondAccepted,
			reason:   string(acceptedReason),
			message:  acceptedMsg,
		},
		programmed: condEntry{
			condType: gatewayCondProgrammed,
			reason:   string(programmedReason),
			message:  programmedMsg,
		},
	}
}

// SetAcceptedError sets an error on the Accepted condition.
func (g *GatewayConditionSet) SetAcceptedError(reason GatewayAcceptedReason, message string) {
	g.accepted.err = &condEntryError{reason: string(reason), message: message}
}

// SetProgrammedError sets an error on the Programmed condition.
func (g *GatewayConditionSet) SetProgrammedError(reason GatewayProgrammedReason, message string) {
	g.programmed.err = &condEntryError{reason: string(reason), message: message}
}

// SetProgrammedMessage updates the success message on the Programmed condition (no-op if error is set).
func (g *GatewayConditionSet) SetProgrammedMessage(msg string) {
	g.programmed.message = msg
}

// Build writes conditions into existing and returns the result.
func (g *GatewayConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	return applyAll(generation, existing, []condEntry{g.accepted, g.programmed})
}

// ---------------------------------------------------------------------------
// ListenerConditionSet — Accepted + Programmed + Conflicted + ResolvedRefs
// ---------------------------------------------------------------------------

type ListenerConditionSet struct {
	accepted     condEntry
	programmed   condEntry
	conflicted   condEntry // negative polarity: default StatusFalse
	resolvedRefs condEntry
}

const (
	listenerCondAccepted     = "Accepted"
	listenerCondProgrammed   = "Programmed"
	listenerCondConflicted   = "Conflicted"
	listenerCondResolvedRefs = "ResolvedRefs"
)

// NewListenerConditionSet creates a set pre-populated with the default success state.
func NewListenerConditionSet(
	acceptedReason ListenerAcceptedReason, acceptedMsg string,
	programmedReason ListenerProgrammedReason, programmedMsg string,
	conflictedReason ListenerConflictedReason, conflictedMsg string,
	resolvedRefsReason ListenerResolvedRefsReason, resolvedRefsMsg string,
) ListenerConditionSet {
	return ListenerConditionSet{
		accepted: condEntry{
			condType: listenerCondAccepted,
			reason:   string(acceptedReason),
			message:  acceptedMsg,
		},
		programmed: condEntry{
			condType: listenerCondProgrammed,
			reason:   string(programmedReason),
			message:  programmedMsg,
		},
		conflicted: condEntry{
			condType: listenerCondConflicted,
			reason:   string(conflictedReason),
			message:  conflictedMsg,
			status:   StatusFalse, // negative polarity
		},
		resolvedRefs: condEntry{
			condType: listenerCondResolvedRefs,
			reason:   string(resolvedRefsReason),
			message:  resolvedRefsMsg,
		},
	}
}

// SetAcceptedError sets an error on the Accepted condition.
func (l *ListenerConditionSet) SetAcceptedError(reason ListenerAcceptedReason, message string) {
	l.accepted.err = &condEntryError{reason: string(reason), message: message}
}

// SetProgrammedError sets an error on the Programmed condition.
func (l *ListenerConditionSet) SetProgrammedError(reason ListenerProgrammedReason, message string) {
	l.programmed.err = &condEntryError{reason: string(reason), message: message}
}

// SetConflictedError sets an error on the Conflicted condition (inverts to StatusTrue).
func (l *ListenerConditionSet) SetConflictedError(reason ListenerConflictedReason, message string) {
	l.conflicted.err = &condEntryError{reason: string(reason), message: message}
}

// SetResolvedRefsError sets an error on the ResolvedRefs condition.
func (l *ListenerConditionSet) SetResolvedRefsError(reason ListenerResolvedRefsReason, message string) {
	l.resolvedRefs.err = &condEntryError{reason: string(reason), message: message}
}

// SetResolvedRefsReason overwrites the success reason/message (e.g. for InvalidRouteKinds set on success path).
func (l *ListenerConditionSet) SetResolvedRefsReason(reason ListenerResolvedRefsReason, status metav1.ConditionStatus, message string) {
	l.resolvedRefs.reason = string(reason)
	l.resolvedRefs.status = status
	l.resolvedRefs.message = message
}

// Build writes conditions into existing and returns the result.
func (l *ListenerConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	return applyAll(generation, existing, []condEntry{
		l.accepted, l.programmed, l.conflicted, l.resolvedRefs,
	})
}

// ---------------------------------------------------------------------------
// RouteConditionSet — Accepted + ResolvedRefs + optional ResolvedWaypoints
// ---------------------------------------------------------------------------

type RouteConditionSet struct {
	accepted          condEntry
	resolvedRefs      condEntry
	resolvedWaypoints *condEntry // optional; only emitted when non-nil
}

const (
	routeCondAccepted          = "Accepted"
	routeCondResolvedRefs      = "ResolvedRefs"
	routeCondResolvedWaypoints = "ResolvedWaypoints"
)

// NewRouteConditionSet creates a set pre-populated with the default success state.
func NewRouteConditionSet(
	acceptedReason RouteAcceptedReason, acceptedMsg string,
	resolvedRefsReason RouteResolvedRefsReason, resolvedRefsMsg string,
) RouteConditionSet {
	return RouteConditionSet{
		accepted: condEntry{
			condType: routeCondAccepted,
			reason:   string(acceptedReason),
			message:  acceptedMsg,
		},
		resolvedRefs: condEntry{
			condType: routeCondResolvedRefs,
			reason:   string(resolvedRefsReason),
			message:  resolvedRefsMsg,
		},
	}
}

// SetAcceptedError sets an error on the Accepted condition.
func (r *RouteConditionSet) SetAcceptedError(reason RouteAcceptedReason, message string) {
	r.accepted.err = &condEntryError{reason: string(reason), message: message}
}

// SetResolvedRefsError sets an error on the ResolvedRefs condition.
func (r *RouteConditionSet) SetResolvedRefsError(reason RouteResolvedRefsReason, message string) {
	r.resolvedRefs.err = &condEntryError{reason: string(reason), message: message}
}

// EnableResolvedWaypoints adds the ResolvedWaypoints condition with a default success state.
func (r *RouteConditionSet) EnableResolvedWaypoints(reason RouteResolvedWaypointsReason, message string) {
	e := condEntry{
		condType: routeCondResolvedWaypoints,
		reason:   string(reason),
		message:  message,
	}
	r.resolvedWaypoints = &e
}

// SetResolvedWaypointsMessage updates the message on the ResolvedWaypoints condition.
// EnableResolvedWaypoints must have been called first.
func (r *RouteConditionSet) SetResolvedWaypointsMessage(message string) {
	if r.resolvedWaypoints != nil {
		r.resolvedWaypoints.message = message
	}
}

// Build writes conditions into existing and returns the result.
func (r *RouteConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	entries := []condEntry{r.accepted, r.resolvedRefs}
	if r.resolvedWaypoints != nil {
		entries = append(entries, *r.resolvedWaypoints)
	}
	return applyAll(generation, existing, entries)
}

// ---------------------------------------------------------------------------
// PolicyConditionSet — Accepted + optional ResolvedRefs (BackendTLSPolicy)
// ---------------------------------------------------------------------------

type PolicyConditionSet struct {
	accepted     condEntry
	resolvedRefs *condEntry // optional
}

const (
	policyCondAccepted     = "Accepted"
	policyCondResolvedRefs = "ResolvedRefs"
)

// NewPolicyConditionSet creates a set with only the Accepted condition.
func NewPolicyConditionSet(acceptedReason PolicyAcceptedReason, acceptedMsg string) PolicyConditionSet {
	return PolicyConditionSet{
		accepted: condEntry{
			condType: policyCondAccepted,
			reason:   string(acceptedReason),
			message:  acceptedMsg,
		},
	}
}

// EnableResolvedRefs adds the ResolvedRefs condition with a default success state.
func (p *PolicyConditionSet) EnableResolvedRefs(reason BackendTLSResolvedRefsReason, message string) {
	e := condEntry{
		condType: policyCondResolvedRefs,
		reason:   string(reason),
		message:  message,
	}
	p.resolvedRefs = &e
}

// SetAcceptedError sets an error on the Accepted condition.
func (p *PolicyConditionSet) SetAcceptedError(reason PolicyAcceptedReason, message string) {
	p.accepted.err = &condEntryError{reason: string(reason), message: message}
}

// AppendAcceptedMessage appends text to the Accepted condition's success message.
func (p *PolicyConditionSet) AppendAcceptedMessage(extra string) {
	p.accepted.message += extra
}

// SetResolvedRefsError sets an error on the ResolvedRefs condition.
// EnableResolvedRefs must have been called first.
func (p *PolicyConditionSet) SetResolvedRefsError(reason BackendTLSResolvedRefsReason, message string) {
	if p.resolvedRefs != nil {
		p.resolvedRefs.err = &condEntryError{reason: string(reason), message: message}
	}
}

// Build writes conditions into existing and returns the result.
func (p *PolicyConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	entries := []condEntry{p.accepted}
	if p.resolvedRefs != nil {
		entries = append(entries, *p.resolvedRefs)
	}
	return applyAll(generation, existing, entries)
}

// ---------------------------------------------------------------------------
// InferencePoolConditionSet — Accepted + ResolvedRefs
// ---------------------------------------------------------------------------

type InferencePoolConditionSet struct {
	accepted     condEntry
	resolvedRefs condEntry
}

const (
	inferencePoolCondAccepted     = "Accepted"
	inferencePoolCondResolvedRefs = "ResolvedRefs"
)

// NewInferencePoolConditionSet creates a set pre-populated with the default success state.
func NewInferencePoolConditionSet(
	acceptedReason InferencePoolAcceptedReason, acceptedStatus metav1.ConditionStatus, acceptedMsg string,
	resolvedRefsReason InferencePoolResolvedRefsReason, resolvedRefsStatus metav1.ConditionStatus, resolvedRefsMsg string,
) InferencePoolConditionSet {
	return InferencePoolConditionSet{
		accepted: condEntry{
			condType: inferencePoolCondAccepted,
			reason:   string(acceptedReason),
			status:   acceptedStatus,
			message:  acceptedMsg,
		},
		resolvedRefs: condEntry{
			condType: inferencePoolCondResolvedRefs,
			reason:   string(resolvedRefsReason),
			status:   resolvedRefsStatus,
			message:  resolvedRefsMsg,
		},
	}
}

// Build writes conditions into existing and returns the result.
func (ip *InferencePoolConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	return applyAll(generation, existing, []condEntry{ip.accepted, ip.resolvedRefs})
}

// ---------------------------------------------------------------------------
// GatewayClassConditionSet — Accepted only
// ---------------------------------------------------------------------------

type GatewayClassConditionSet struct {
	accepted condEntry
}

const gatewayClassCondAccepted = "Accepted"

// NewGatewayClassConditionSet creates a set pre-populated with the default success state.
func NewGatewayClassConditionSet(reason GatewayClassAcceptedReason, message string) GatewayClassConditionSet {
	return GatewayClassConditionSet{
		accepted: condEntry{
			condType: gatewayClassCondAccepted,
			reason:   string(reason),
			message:  message,
		},
	}
}

// Build writes conditions into existing and returns the result.
func (gc *GatewayClassConditionSet) Build(generation int64, existing []metav1.Condition) []metav1.Condition {
	return applyAll(generation, existing, []condEntry{gc.accepted})
}
