FROM bash:latest

# Task metadata
LABEL io.oplet.task.meta.name="Hello World !"
LABEL io.oplet.task.meta.description="A basic example of an Oplet task"
LABEL io.oplet.task.meta.author="Bornholm"
LABEL io.oplet.task.meta.url="https://github.com/bornholm/oplet/misc/tasks/hello-world"

# Environment variable inputs
LABEL io.oplet.task.inputs.text_env.type="text"
LABEL io.oplet.task.inputs.text_env.description="A dummy 'text' variable"
LABEL io.oplet.task.inputs.text_env.required="true"

LABEL io.oplet.task.inputs.number_env.type="number"
LABEL io.oplet.task.inputs.number_env.description="A dummy 'number' variable"

LABEL io.oplet.task.inputs.secret_env.type="secret"
LABEL io.oplet.task.inputs.secret_env.description="A dummy 'secret' variable"

LABEL io.oplet.task.inputs.bool_env.type="boolean"
LABEL io.oplet.task.inputs.bool_env.description="A dummy 'boolean' variable"

LABEL io.oplet.task.inputs.dummy_file.type="file"
LABEL io.oplet.task.inputs.dummy_file.description="A dummy file to provide to the task"

LABEL io.oplet.task.inputs.must_fail.label="Must fail ?"
LABEL io.oplet.task.inputs.must_fail.type="boolean"
LABEL io.oplet.task.inputs.must_fail.description="Force the task to fail"

# Configuration parameters
LABEL io.oplet.task.config.max_rows.type="text"
LABEL io.oplet.task.config.max_rows.description="Dummy configuration text var"

# Declare cached directory
VOLUME /cache

COPY --chmod=+x run.sh /usr/local/bin/run.sh

CMD ["/usr/local/bin/run.sh"]