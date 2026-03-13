apiVersion: v1
kind: Secret
metadata:
  name: {{ include "ketches.secretName" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
type: Opaque
stringData:
  jwt-secret: {{ required "values.config.jwtSecret is required" .Values.config.jwtSecret | quote }}
  {{- if .Values.config.dbSource }}
  db-source: {{ .Values.config.dbSource | quote }}
  {{- else if ne .Values.config.dbDriver "sqlite" }}
  db-password: {{ include "ketches.database.password" . | quote }}
  {{- end }}
