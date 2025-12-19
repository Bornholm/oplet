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