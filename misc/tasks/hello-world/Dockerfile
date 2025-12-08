FROM bash:latest

# Task metadata
LABEL io.oplet.task.meta.name="Hello World !"
LABEL io.oplet.task.meta.description="A basic example of an Oplet task"
LABEL io.oplet.task.meta.author="Bornholm"
LABEL io.oplet.task.meta.url="https://github.com/bornholm/oplet/misc/tasks/hello-world"

# Environment variable inputs
LABEL io.oplet.task.inputs.text_env.type="env"
LABEL io.oplet.task.inputs.text_env.value_type="text"
LABEL io.oplet.task.inputs.text_env.description="A dummy 'text' variable"
LABEL io.oplet.task.inputs.text_env.required="true"

LABEL io.oplet.task.inputs.number_env.type="env"
LABEL io.oplet.task.inputs.number_env.value_type="number"
LABEL io.oplet.task.inputs.number_env.description="A dummy 'number' variable"
LABEL io.oplet.task.inputs.number_env.required="false"

# File inputs
LABEL io.oplet.task.inputs.dummy_file.type="file"
LABEL io.oplet.task.inputs.dummy_file.value_type="file"
LABEL io.oplet.task.inputs.dummy_file.description="A dummy file to provide to the task"
LABEL io.oplet.task.inputs.dummy_file.required="false"

# Configuration parameters
LABEL io.oplet.task.config.max_rows.type="env"
LABEL io.oplet.task.config.max_rows.value_type="number"
LABEL io.oplet.task.config.max_rows.description="Maximum number of rows to process"
LABEL io.oplet.task.config.max_rows.required="false"

LABEL io.oplet.task.config.timeout.type="env"
LABEL io.oplet.task.config.timeout.value_type="number"
LABEL io.oplet.task.config.timeout.description="Processing timeout in seconds"
LABEL io.oplet.task.config.timeout.required="false"

COPY --chmod=+x run.sh /usr/local/bin/run.sh

CMD ["/usr/local/bin/run.sh"]