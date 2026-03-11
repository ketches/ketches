{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "ketches.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.className }}
  ingressClassName: {{ .Values.ingress.className | quote }}
  {{- end }}
  {{- with .Values.ingress.tls }}
  tls:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ .host | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path | quote }}
            pathType: {{ default "Prefix" .pathType }}
            backend:
              service:
                {{- if eq .backendService "api" }}
                name: {{ include "ketches.api.fullname" $ | quote }}
                port:
                  number: {{ $.Values.api.service.port }}
                {{- else if eq .backendService "ui" }}
                name: {{ include "ketches.ui.fullname" $ | quote }}
                port:
                  number: {{ $.Values.ui.service.port }}
                {{- else }}
                {{- fail "values.ingress.hosts[].paths[].backendService must be 'api' or 'ui'" }}
                {{- end }}
          {{- end }}
    {{- end }}
{{- end }}
