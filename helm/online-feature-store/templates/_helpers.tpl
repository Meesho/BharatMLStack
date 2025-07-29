{{/*
Expand the name of the chart.
*/}}
{{- define "online-feature-store.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "online-feature-store.fullname" -}}
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
{{- define "online-feature-store.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "online-feature-store.labels" -}}
helm.sh/chart: {{ include "online-feature-store.chart" . }}
{{ include "online-feature-store.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/component: feature-store
app.kubernetes.io/part-of: bharatml-stack
{{- with .Values.labels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "online-feature-store.selectorLabels" -}}
app.kubernetes.io/name: {{ include "online-feature-store.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "online-feature-store.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "online-feature-store.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image pull secret names
*/}}
{{- define "online-feature-store.imagePullSecrets" -}}
{{- with .Values.global.imagePullSecrets }}
{{- range . }}
- name: {{ . }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create a default network policy name
*/}}
{{- define "online-feature-store.networkPolicyName" -}}
{{- printf "%s-netpol" (include "online-feature-store.fullname" .) }}
{{- end }}

{{/*
Create a default service monitor name
*/}}
{{- define "online-feature-store.serviceMonitorName" -}}
{{- printf "%s-metrics" (include "online-feature-store.fullname" .) }}
{{- end }}

{{/*
Create a default HPA name
*/}}
{{- define "online-feature-store.hpaName" -}}
{{- printf "%s-hpa" (include "online-feature-store.fullname" .) }}
{{- end }}

{{/*
Create a default VPA name
*/}}
{{- define "online-feature-store.vpaName" -}}
{{- printf "%s-vpa" (include "online-feature-store.fullname" .) }}
{{- end }}

{{/*
Create a default PDB name
*/}}
{{- define "online-feature-store.pdbName" -}}
{{- printf "%s-pdb" (include "online-feature-store.fullname" .) }}
{{- end }}

{{/*
Create ingress hostname
*/}}
{{- define "online-feature-store.ingressHost" -}}
{{- if .Values.ingress.hosts }}
{{- range .Values.ingress.hosts }}
{{- .host }}
{{- end }}
{{- else }}
{{- printf "%s.%s" (include "online-feature-store.name" .) "local" }}
{{- end }}
{{- end }}

{{/*
Create gateway hostname
*/}}
{{- define "online-feature-store.gatewayHost" -}}
{{- if .Values.gateway.hosts }}
{{- range .Values.gateway.hosts }}
{{- . }}
{{- end }}
{{- else }}
{{- printf "%s.%s" (include "online-feature-store.name" .) "local" }}
{{- end }}
{{- end }}
