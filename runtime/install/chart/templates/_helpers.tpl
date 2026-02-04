{{/*
Copyright AGNTCY Contributors (https://github.com/agntcy)
SPDX-License-Identifier: Apache-2.0
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "runtime.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "runtime.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "runtime.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "runtime.labels" -}}
helm.sh/chart: {{ include "runtime.chart" . }}
{{ include "runtime.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "runtime.selectorLabels" -}}
app.kubernetes.io/name: {{ include "runtime.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* ============================================================================
    Discovery Component Helpers
    ============================================================================ */}}

{{/*
Discovery fullname
*/}}
{{- define "runtime.discovery.fullname" -}}
{{- printf "%s-discovery" (include "runtime.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Discovery labels
*/}}
{{- define "runtime.discovery.labels" -}}
helm.sh/chart: {{ include "runtime.chart" . }}
{{ include "runtime.discovery.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Discovery selector labels
*/}}
{{- define "runtime.discovery.selectorLabels" -}}
app.kubernetes.io/name: {{ include "runtime.name" . }}-discovery
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: discovery
{{- end }}

{{/*
Create the name of the discovery service account to use
*/}}
{{- define "runtime.discovery.serviceAccountName" -}}
{{- if .Values.discovery.serviceAccount.create }}
{{- default (include "runtime.discovery.fullname" .) .Values.discovery.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.discovery.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* ============================================================================
    Server Component Helpers
    ============================================================================ */}}

{{/*
Server fullname
*/}}
{{- define "runtime.server.fullname" -}}
{{- printf "%s-server" (include "runtime.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Server labels
*/}}
{{- define "runtime.server.labels" -}}
helm.sh/chart: {{ include "runtime.chart" . }}
{{ include "runtime.server.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Server selector labels
*/}}
{{- define "runtime.server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "runtime.name" . }}-server
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: server
{{- end }}

{{/*
Create the name of the server service account to use
*/}}
{{- define "runtime.server.serviceAccountName" -}}
{{- if .Values.server.serviceAccount.create }}
{{- default (include "runtime.server.fullname" .) .Values.server.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.server.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* ============================================================================
    Etcd Component Helpers
    ============================================================================ */}}

{{/*
Etcd fullname
*/}}
{{- define "runtime.etcd.fullname" -}}
{{- printf "%s-etcd" (include "runtime.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Get etcd host - returns the chart's etcd service name if enabled, otherwise the configured host
*/}}
{{- define "runtime.etcd.host" -}}
{{- if .Values.etcd.enabled }}
{{- include "runtime.etcd.fullname" . }}
{{- else }}
{{- .host | default "etcd" }}
{{- end }}
{{- end }}
