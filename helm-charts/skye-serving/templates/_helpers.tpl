{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "labels.selector" -}}
app: {{ .Values.namespace }}
{{- end -}}

{{- define "labels.primary-selector" -}}
app: {{ .Values.namespace }}-primary
{{- end -}}

{{- define "labels.common" -}}
{{ template "labels.selector" . }}
{{- if and .Values.deployment .Values.deployment.image }}
version: {{ .Values.deployment.image.tag }}
{{- end }}
env: {{ .Values.labels.env }}
team: {{ .Values.labels.team }}
bu: {{ .Values.labels.bu }}
service: {{ .Values.applicationName }}
priority: {{ .Values.labels.priority }}
priority_v2: {{ .Values.labels.priority_v2 | default "cp3" }}
primary_owner: {{ .Values.labels.primary_owner | default .Values.labels.team }}
secondary_owner: {{ .Values.labels.secondary_owner | default .Values.labels.team }}
service_type: {{ .Values.labels.service_type | default "" | replace "," "-" | quote }}
{{- end -}}

{{- define "labels.chart" -}}
chart: "{{ .Chart.Name }}-{{ .Chart.Version }}"
release: {{ .Release.Name | quote }}
heritage: {{ .Release.Service | quote }}
{{- end -}}

{{/*
Renders a value that contains template.
Usage:
{{ include "application.tplvalues.render" ( dict "value" .Values.path.to.the.Value "context" $) }}
*/}}

{{- define "application.tplvalues.render" -}}
    {{- if typeIs "string" .value }}
        {{- tpl .value .context }}
    {{- else }}
        {{- tpl (.value | toYaml) .context }}
    {{- end }}
{{- end -}}

{{- define "canary.promURL" -}}
{{- if .Values.canary.promURL }}
{{- .Values.canary.promURL }}
{{- else if eq .Values.labels.env "prod" }}
prod-ops-metricsui.example.com/select/100/
{{- else }}
https://sb-ops-metricsui.example.com/select/100/
{{- end }}
{{- end -}}
