apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "ketches.api.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: api
spec:
  replicas: {{ .Values.api.replicaCount }}
  selector:
    matchLabels:
      {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "api") | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/secret: {{ include (print $.Template.BasePath "/secret.yaml.tpl") . | sha256sum | quote }}
        {{- with .Values.api.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "api") | nindent 8 }}
        {{- with .Values.api.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: ketches-api
          image: "{{ .Values.api.image.repository }}:{{ .Values.api.image.tag }}"
          imagePullPolicy: {{ .Values.api.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.api.containerPort }}
              protocol: TCP
          env:
            - name: PORT
              value: {{ .Values.config.port | quote }}
            - name: LOG_LEVEL
              value: {{ .Values.config.logLevel | quote }}
            - name: DB_DRIVER
              value: {{ .Values.config.dbDriver | quote }}
            - name: DB_AUTO_MIGRATE
              value: {{ .Values.config.dbAutoMigrate | quote }}
            {{- if .Values.config.dbSource }}
            - name: DB_SOURCE
              valueFrom:
                secretKeyRef:
                  name: {{ include "ketches.secretName" . | quote }}
                  key: db-source
            {{- else }}
            - name: DB_NAME
              value: {{ include "ketches.database.name" . | quote }}
            {{- if ne .Values.config.dbDriver "sqlite" }}
            - name: DB_HOST
              value: {{ include "ketches.database.host" . | quote }}
            - name: DB_PORT
              value: {{ include "ketches.database.port" . | quote }}
            - name: DB_USERNAME
              value: {{ include "ketches.database.username" . | quote }}
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "ketches.secretName" . | quote }}
                  key: db-password
            {{- end }}
            {{- if eq .Values.config.dbDriver "postgres" }}
            - name: DB_SSLMODE
              value: {{ .Values.config.dbSSLMode | quote }}
            {{- end }}
            {{- end }}
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: {{ include "ketches.secretName" . | quote }}
                  key: jwt-secret
            - name: CORS_ALLOWED_ORIGINS
              value: {{ .Values.config.corsAllowedOrigins | quote }}
            {{- range .Values.api.extraEnv }}
            - name: {{ .name }}
              {{- if hasKey . "valueFrom" }}
              valueFrom:
                {{- toYaml .valueFrom | nindent 16 }}
              {{- else }}
              value: {{ .value | quote }}
              {{- end }}
            {{- end }}
          {{- if .Values.api.livenessProbe.enabled }}
          livenessProbe:
            httpGet:
              path: {{ .Values.api.livenessProbe.path | quote }}
              port: http
            initialDelaySeconds: {{ .Values.api.livenessProbe.initialDelaySeconds }}
            periodSeconds: {{ .Values.api.livenessProbe.periodSeconds }}
            timeoutSeconds: {{ .Values.api.livenessProbe.timeoutSeconds }}
            failureThreshold: {{ .Values.api.livenessProbe.failureThreshold }}
          {{- end }}
          {{- if .Values.api.readinessProbe.enabled }}
          readinessProbe:
            httpGet:
              path: {{ .Values.api.readinessProbe.path | quote }}
              port: http
            initialDelaySeconds: {{ .Values.api.readinessProbe.initialDelaySeconds }}
            periodSeconds: {{ .Values.api.readinessProbe.periodSeconds }}
            timeoutSeconds: {{ .Values.api.readinessProbe.timeoutSeconds }}
            failureThreshold: {{ .Values.api.readinessProbe.failureThreshold }}
          {{- end }}
          resources:
            {{- toYaml .Values.api.resources | nindent 12 }}
      {{- with .Values.api.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.api.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.api.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
