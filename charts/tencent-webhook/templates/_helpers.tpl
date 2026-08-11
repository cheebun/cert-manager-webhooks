{{/* vim: set filetype=mustache: */}}
{{- define "tencent-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tencent-webhook.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "tencent-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tencent-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "tencent-webhook.fullname" .) }}
{{- end -}}

{{- define "tencent-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "tencent-webhook.fullname" .) }}
{{- end -}}

{{- define "tencent-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "tencent-webhook.fullname" .) }}
{{- end -}}

{{- define "tencent-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "tencent-webhook.fullname" .) }}
{{- end -}}
