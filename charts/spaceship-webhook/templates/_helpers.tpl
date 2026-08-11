{{/* vim: set filetype=mustache: */}}
{{- define "spaceship-webhook.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "spaceship-webhook.fullname" -}}
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

{{- define "spaceship-webhook.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "spaceship-webhook.selfSignedIssuer" -}}
{{ printf "%s-selfsign" (include "spaceship-webhook.fullname" .) }}
{{- end -}}

{{- define "spaceship-webhook.rootCAIssuer" -}}
{{ printf "%s-ca" (include "spaceship-webhook.fullname" .) }}
{{- end -}}

{{- define "spaceship-webhook.rootCACertificate" -}}
{{ printf "%s-ca" (include "spaceship-webhook.fullname" .) }}
{{- end -}}

{{- define "spaceship-webhook.servingCertificate" -}}
{{ printf "%s-webhook-tls" (include "spaceship-webhook.fullname" .) }}
{{- end -}}
