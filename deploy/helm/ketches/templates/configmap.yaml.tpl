apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "ketches.configMapName" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
data:
  PORT: {{ .Values.config.port | quote }}
  LOG_LEVEL: {{ .Values.config.logLevel | quote }}
  DB_DRIVER: {{ .Values.config.dbDriver | quote }}
  CORS_ALLOWED_ORIGINS: {{ .Values.config.corsAllowedOrigins | quote }}
  BACKEND_URL: {{ include "ketches.ui.backendUrl" . | quote }}
