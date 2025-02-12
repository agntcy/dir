apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: {{ generate_name }}-
spec:
  entrypoint: agent-task-execution
  imagePullSecrets:
{% for image_pull_secret in image_pull_secrets %}
    - name: {{ image_pull_secret }}
{% endfor %}

  ttlStrategy:
    secondsAfterCompletion: 60
    secondsAfterSuccess: 60
    secondsAfterFailure: 600

  templates:
    - name: agent-task-execution
      steps:
        - - name: agent
            template: agent-runner
            arguments:
              parameters:
                - name: program
                  value: {{ program }}
                - name: input-message
                  value: |
                    {{ input_message | indent(12) }}
                - name: output-file
                  value: /tmp/result.json

    - name: agent-runner
      inputs:
        parameters:
          - name: input-message
          - name: output-file
          - name: program
      container:
        image: {{ agent_runner_image }}
        command:
          - python
        env:
{% for env_var_name, env_var_value in additional_env_vars.items() %}
          - name: {{ env_var_name }}
            value: {{ env_var_value }}
{% endfor %}
{% for env_var_name, env_var_secret_key in additional_env_vars_from_secret.items() %}
          - name: {{ env_var_name }}
            valueFrom:
              secretKeyRef:
                name: {{ env_var_secret_key[0] }}
                key: {{ env_var_secret_key[1] }}
{% endfor %}
        args:
          - -m
          - '{-inputs.parameters.program-}'
          - --input-json
          - '{-inputs.parameters.input-message-}'
          - --output-file
          - '{-inputs.parameters.output-file-}'
      outputs:
        parameters:
          - name: result
            valueFrom:
              path: "{-inputs.parameters.output-file-}"
