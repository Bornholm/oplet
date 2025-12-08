TASK_VERSION ?= latest
DOCKER_REPOSITORY ?= docker.io/bornholm
AVAILABLE_TASKS ?= $(shell find ./misc/tasks/ -mindepth 1 -maxdepth 1 -type d -printf '%P\n')

task-build-images: $(foreach task_name,$(AVAILABLE_TASKS),task-build-image-$(task_name))

task-build-image-%:
	docker build \
		-t $(DOCKER_REPOSITORY)/oplet-$*-task:$(TASK_VERSION) \
		./misc/tasks/$*


task-release-images: task-build-images $(foreach task_name,$(AVAILABLE_TASKS),task-release-image-$(task_name))

task-release-image-%:
	docker push $(DOCKER_REPOSITORY)/oplet-$*-task:$(TASK_VERSION)