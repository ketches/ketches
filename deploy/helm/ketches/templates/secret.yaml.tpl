{{- $secretName := include "ketches.secretName" . -}}
{{- $bootstrapAdminPassword := .Values.config.bootstrapAdminPassword | default "" -}}
{{- if not $bootstrapAdminPassword -}}
  {{- $existingSecret := lookup "v1" "Secret" .Release.Namespace $secretName -}}
  {{- if and $existingSecret $existingSecret.data -}}
    {{- $existingPassword := index $existingSecret.data "bootstrap-admin-password" | default "" -}}
    {{- if $existingPassword -}}
      {{- $bootstrapAdminPassword = $existingPassword | b64dec -}}
    {{- end -}}
  {{- end -}}
  {{- if not $bootstrapAdminPassword -}}
    {{- $bootstrapAdminPassword = randAlphaNum 32 -}}
  {{- end -}}
{{- end -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{ $secretName | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
type: Opaque
stringData:
  jwt-secret: {{ required "values.config.jwtSecret is required" .Values.config.jwtSecret | quote }}
  secret-encryption-key: {{ required "values.config.secretEncryptionKey is required" .Values.config.secretEncryptionKey | quote }}
  bootstrap-admin-username: {{ .Values.config.bootstrapAdminUsername | quote }}
  bootstrap-admin-password: {{ $bootstrapAdminPassword | quote }}
  smtp-password: {{ .Values.config.smtpPassword | quote }}
  {{- if .Values.config.dbSource }}
  db-source: {{ .Values.config.dbSource | quote }}
  {{- else }}
  db-password: {{ include "ketches.database.password" . | quote }}
  {{- end }}
