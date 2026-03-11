apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "ketches.ui.fullname" . | quote }}
  labels:
    {{- include "ketches.labels" . | nindent 4 }}
    app.kubernetes.io/component: ui
spec:
  replicas: {{ .Values.ui.replicaCount }}
  selector:
    matchLabels:
      {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "ui") | nindent 6 }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml.tpl") . | sha256sum | quote }}
        {{- with .Values.ui.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      labels:
        {{- include "ketches.componentSelectorLabels" (dict "context" . "component" "ui") | nindent 8 }}
        {{- with .Values.ui.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      containers:
        - name: ketches-ui
          image: "{{ .Values.ui.image.repository }}:{{ .Values.ui.image.tag }}"
          imagePullPolicy: {{ .Values.ui.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.ui.containerPort }}
              protocol: TCP
          env:
            - name: BACKEND_URL
              valueFrom:
                configMapKeyRef:
                  name: {{ include "ketches.configMapName" . | quote }}
                  key: BACKEND_URL
            {{- range .Values.ui.extraEnv }}
            - name: {{ .name }}
              {{- if hasKey . "valueFrom" }}
              valueFrom:
                {{- toYaml .valueFrom | nindent 16 }}
              {{- else }}
              value: {{ .value | quote }}
              {{- end }}
            {{- end }}
          resources:
            {{- toYaml .Values.ui.resources | nindent 12 }}
      {{- with .Values.ui.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.ui.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.ui.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
