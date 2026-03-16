{{/* Chart name. */}}
{{- define "ketches.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Fully qualified app name. */}}
{{- define "ketches.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "ketches.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Chart + version label value. */}}
{{- define "ketches.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common labels. */}}
{{- define "ketches.labels" -}}
helm.sh/chart: {{ include "ketches.chart" . }}
app.kubernetes.io/name: {{ include "ketches.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Labels used in selectors. */}}
{{- define "ketches.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ketches.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Component-scoped selector labels. */}}
{{- define "ketches.componentSelectorLabels" -}}
{{ include "ketches.selectorLabels" .context }}
app.kubernetes.io/component: {{ .component }}
{{- end -}}

{{/* Component names. */}}
{{- define "ketches.api.fullname" -}}
{{- printf "%s-api" (include "ketches.fullname" .) -}}
{{- end -}}

{{- define "ketches.ui.fullname" -}}
{{- printf "%s-ui" (include "ketches.fullname" .) -}}
{{- end -}}

{{- define "ketches.postgres.fullname" -}}
{{- printf "%s-postgres" (include "ketches.fullname" .) -}}
{{- end -}}

{{- define "ketches.configMapName" -}}
{{- printf "%s-config" (include "ketches.fullname" .) -}}
{{- end -}}

{{- define "ketches.secretName" -}}
{{- printf "%s-secrets" (include "ketches.fullname" .) -}}
{{- end -}}

{{/* Database connection value helpers. */}}
{{- define "ketches.database.host" -}}
{{- if .Values.postgres.enabled -}}
{{- include "ketches.postgres.fullname" . -}}
{{- else if .Values.config.dbHost -}}
{{- .Values.config.dbHost -}}
{{- else -}}
{{- fail "values.config.dbHost must be set when values.postgres.enabled is false and values.config.dbSource is empty" -}}
{{- end -}}
{{- end -}}

{{- define "ketches.database.port" -}}
{{- if .Values.postgres.enabled -}}
{{- printf "%v" (int .Values.postgres.service.port) -}}
{{- else if .Values.config.dbPort -}}
{{- .Values.config.dbPort -}}
{{- else if eq .Values.config.dbDriver "mysql" -}}
{{- printf "3306" -}}
{{- else -}}
{{- printf "5432" -}}
{{- end -}}
{{- end -}}

{{- define "ketches.database.name" -}}
{{- if .Values.postgres.enabled -}}
{{- .Values.postgres.auth.database -}}
{{- else if .Values.config.dbName -}}
{{- .Values.config.dbName -}}
{{- else -}}
{{- fail "values.config.dbName must be set when values.postgres.enabled is false and values.config.dbSource is empty" -}}
{{- end -}}
{{- end -}}

{{- define "ketches.database.username" -}}
{{- if .Values.postgres.enabled -}}
{{- .Values.postgres.auth.username -}}
{{- else if .Values.config.dbUsername -}}
{{- .Values.config.dbUsername -}}
{{- else if eq .Values.config.dbDriver "mysql" -}}
{{- printf "root" -}}
{{- else -}}
{{- fail "values.config.dbUsername must be set when values.postgres.enabled is false and values.config.dbSource is empty" -}}
{{- end -}}
{{- end -}}

{{- define "ketches.database.password" -}}
{{- if .Values.postgres.enabled -}}
{{- .Values.postgres.auth.password -}}
{{- else if .Values.config.dbPassword -}}
{{- .Values.config.dbPassword -}}
{{- else -}}
{{- fail "values.config.dbPassword must be set when values.postgres.enabled is false and values.config.dbSource is empty" -}}
{{- end -}}
{{- end -}}

{{/* UI backend URL generation. */}}
{{- define "ketches.ui.backendUrl" -}}
{{- if .Values.config.backendUrl -}}
{{- .Values.config.backendUrl -}}
{{- else -}}
{{- printf "http://%s:%v" (include "ketches.api.fullname" .) (int .Values.api.service.port) -}}
{{- end -}}
{{- end -}}
