apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ketches.secretName" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
type: Opaque
stringData:
  jwt-secret: {{ required "values.config.jwtSecret is required" .Values.config.jwtSecret | quote }}
  secret-encryption-key: {{ required "values.config.secretEncryptionKey is required" .Values.config.secretEncryptionKey | quote }}
  {{- if .Values.config.dbSource }}
  db-source: {{ .Values.config.dbSource | quote }}
  {{- else }}
  db-password: {{ include "ketches.database.password" . | quote }}
  {{- end }}
