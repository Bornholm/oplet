# Creating an Oplet Task

Oplet transforms any standard Docker image into a "Task" executable via a web interface. The magic happens thanks to **Dockerfile LABELS**, which describe to Oplet how to generate the input form and how to run the container.

In this tutorial, we will create a tool that resizes and optimizes images (JPG/PNG) using **ImageMagick**.

> The task implemented in this tutorial is available as `docker.io/bornholm/oplet-image-optimizer-task:latest`.

## 1. The Script Logic

Before dealing with Docker, we need a script to do the actual work. Oplet injects uploaded files into `/oplet/inputs` and passes parameters as environment variables.

Create a file named `optimize.sh`:

```bash
#!/bin/sh
set -e

# Oplet places "file" type inputs in this directory
SOURCE_IMAGE="/oplet/inputs/SOURCE_IMAGE"

# Retrieving environment variables (defined via Oplet Labels)
QUALITY="${IMG_QUALITY:-85}"   # Default value 85
TARGET_WIDTH="${IMG_WIDTH}"    # Target width
GRAYSCALE="${TO_GRAYSCALE}"    # "true" or "false"

# Oplet will automatically collect all files placed in /oplet/outputs
output_path="/oplet/outputs/optimized.jpg"

echo "🚀 Starting optimization for: $FILE_NAME"
echo "ℹ️  Parameters: Quality=$QUALITY, Width=$TARGET_WIDTH, Grayscale=$GRAYSCALE"

# Building the magick command
CMD="magick '$SOURCE_IMAGE' -quality $QUALITY"

if [ -n "$TARGET_WIDTH" ]; then
    CMD="$CMD -resize ${TARGET_WIDTH}x"
fi

if [ "$GRAYSCALE" = "true" ]; then
    CMD="$CMD -colorspace Gray"
fi

CMD="$CMD '$output_path'"

# Execution
eval $CMD

echo "✅ Success! Image optimized: $output_path"
ls -lh "$output_path"
```

---

## 2. The Dockerfile (The Oplet Definition)

This is where everything happens. We will use `io.oplet.task.*` labels to define the user interface.

Create the `Dockerfile`:

```dockerfile
FROM alpine:3.23

# Install dependencies
RUN apk add --no-cache imagemagick bash

# ==========================================
# 1. Task Metadata
# ==========================================
LABEL io.oplet.task.meta.name="Image Optimizer"
LABEL io.oplet.task.meta.description="Optimizes, resizes and converts images via ImageMagick"
LABEL io.oplet.task.meta.author="Bornholm"
LABEL io.oplet.task.meta.url="https://github.com/bornholm/oplet/misc/tasks/image-optimizer"

# ==========================================
# 2. Inputs Definition
# These are the fields the user fills in for every execution
# ==========================================

# > Type FILE
LABEL io.oplet.task.inputs.SOURCE_IMAGE.label="Image"
LABEL io.oplet.task.inputs.SOURCE_IMAGE.type="file"
LABEL io.oplet.task.inputs.SOURCE_IMAGE.description="The original image to process (JPG, PNG)"
LABEL io.oplet.task.inputs.SOURCE_IMAGE.required="true"

# > Type TEXT (Optional max width)
LABEL io.oplet.task.inputs.IMG_WIDTH.label="Target Width (px)"
LABEL io.oplet.task.inputs.IMG_WIDTH.type="text"
LABEL io.oplet.task.inputs.IMG_WIDTH.description="Leave empty to keep original size"

# > Type BOOLEAN (Checkbox)
LABEL io.oplet.task.inputs.TO_GRAYSCALE.label="Convert to B&W?"
LABEL io.oplet.task.inputs.TO_GRAYSCALE.type="boolean"
LABEL io.oplet.task.inputs.TO_GRAYSCALE.description="Check to convert image to grayscale"

# ==========================================
# 3. Configuration Definition (Config)
# These are 'hidden' or 'admin' settings, defined when the task is deployed
# ==========================================

LABEL io.oplet.task.config.IMG_QUALITY.label="Compression Quality"
LABEL io.oplet.task.config.IMG_QUALITY.type="number"
LABEL io.oplet.task.config.IMG_QUALITY.description="Between 1 and 100 (Default: 85)"

# Copy script and set execution permissions
COPY optimize.sh /usr/local/bin/optimize.sh
RUN chmod +x /usr/local/bin/optimize.sh

CMD ["/usr/local/bin/optimize.sh"]
```

---

## 3. Understanding Oplet Concepts

### A. Inputs vs. Config: What's the difference?

- **Inputs (`io.oplet.task.inputs.*`)**: Designed for the **End User**.

  - _When?_ Every time someone clicks "Run Task".
  - _Examples:_ CSV file to process, start date, recipient email.
  - _Result:_ Generates the dynamic HTML form.

- **Config (`io.oplet.task.config.*`)**: Designed for the **Operator/Admin**.
  - _When?_ Set only once, during the import or initial configuration of the task in Oplet.
  - _Examples:_ Slack API Key, Database URL, process niceness.
  - _Result:_ These values are injected transparently into the container; the user does not see them in the execution form.

### B. Supported Field Types

In the `.type="..."` labels, you can use:

1.  `text`: Simple text field.
2.  `number`: Numeric field (HTML `<input type="number">`).
3.  `boolean`: Checkbox (Returns "true" or "false").
4.  `secret`: Masked field (type password), useful for tokens/keys.
5.  `file`: File upload. Oplet places it in the container in the `/oplet/inputs` directory

---

## 4. Build and Test

1.  **Build the image:**

    ```bash
    docker build -t my-registry/oplet-optimizer:v1 .
    ```

2.  **Test Locally (Manual Simulation):**
    You can test your image without Oplet to verify the script works by mimicking Oplet's behavior:

    ```bash
    # Create a fake inputs folder
    mkdir -p /tmp/oplet-test/inputs
    cp my-photo.jpg /tmp/oplet-test/inputs/SOURCE_IMAGE

    # Run the container as Oplet would
    docker run --rm \
        -v /tmp/oplet-test/inputs:/oplet/inputs \
        -v /tmp/oplet-test/outputs:/oplet/outputs \
        -e IMG_QUALITY="50" \
        -e TO_GRAYSCALE="true" \
        my-registry/oplet-optimizer:v1
    ```

3.  **Use in Oplet:**
    Simply load the image `my-registry/oplet-optimizer:v1` into your Oplet instance. The form will be generated automatically.
