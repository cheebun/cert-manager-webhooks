{{/* vim: set filetype=mustache: */}}
{{- define "alidns-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "alidns-webhook.fullname" -}}
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

{{- define "alidns-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "alidns-webhook.labels" -}}
helm.sh/chart: {{ include "alidns-webhook.chart" . }}
{{ include "alidns-webhook.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "alidns-webhook.selectorLabels" -}}
app.kubernetes.io/name: {{ include "alidns-webhook.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "alidns-webhook.imageTag" -}}
{{- if .Values.image.tag }}
{{- .Values.image.tag }}
{{- else -}}
v{{- .Chart.Version }}
{{- end }}
{{- end }}

{{- define "alidns-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "alidns-webhook.fullname" .) }}
{{- end -}}

{{- define "alidns-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "alidns-webhook.fullname" .) }}
{{- end -}}

{{- define "alidns-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "alidns-webhook.fullname" .) }}
{{- end -}}

{{- define "alidns-webhook.servingCertificate" -}}
{{ printf "%s-tls" (include "alidns-webhook.fullname" .) }}
{{- end -}}
