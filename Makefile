GOPROXY_OLD=$(shell go env GOPROXY)
GOPROXY=https://goproxy.io,https://goproxy.cn,direct
install_golangci_lint=(go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
APP_NAME=demo1
OS=
Upx=0
ARG=
StartArg=start
CompileMod=production
Version=0.0.1
LDPackage=github.com/atcharles/glibs/util
LDFlags=-s -w
LDFlags+= -X '$(LDPackage).AppName=$(APP_NAME)'
LDFlags+= -X '$(LDPackage).CompileMod=$(CompileMod)'
LDFlags+= -X '$(LDPackage).Version=$(Version)'
LDFlags+= -X '$(LDPackage).BuildTime=`date '+%Y-%m-%d %H:%M:%S %Z/%p'`'
LDFlags+= -X '$(LDPackage).GoVersion=`go version`'
LDFlags+= -X '$(LDPackage).GitBranch=`git rev-parse --abbrev-ref HEAD`'
LDFlags+= -X '$(LDPackage).GitHash=`git rev-parse HEAD`'
TIDY_CMD=go mod tidy
CGO_ENABLED=0
ifeq (-race,$(findstring -race,$(ARG)))
CGO_ENABLED=1
endif
BuildCMD=CGO_ENABLED=$(CGO_ENABLED) go build $(ARG) -trimpath -v -ldflags="$(LDFlags)" -o
ifeq ($(OS),linux)
	BuildCMD:=GOOS="linux" GOARCH="amd64" $(BuildCMD)
endif
.PHONY:vet lint upgrade commit-init git-push git-pull buildapp clean run
run:buildapp
	@echo "run $(APP_NAME)"
	@./apps/$(APP_NAME)/build/$(APP_NAME).out $(StartArg)
clean:
	@echo "Cleaning..."
	@rm -rf apps/$(APP_NAME)/build/logs
	@rm -rf apps/$(APP_NAME)/build/*.out
	@rm -rf apps/$(APP_NAME)/build/*.log
	@rm -rf apps/$(APP_NAME)/build/$(APP_NAME)
	@go clean ./...
buildapp:clean vet
	@echo "build app $(APP_NAME)"
	@$(BuildCMD) apps/$(APP_NAME)/build/$(APP_NAME).out ./apps/$(APP_NAME)
ifneq ($(Upx), 0)
	@upx -9 apps/$(APP_NAME)/build/$(APP_NAME).out
endif
	@chmod +x apps/$(APP_NAME)/build/$(APP_NAME).out
	@echo "build app done"
vet:
	@echo "Running go vet"
	@go env -w GOPROXY=$(GOPROXY)
	@$(TIDY_CMD)
	@go vet ./...
	@go env -w GOPROXY=$(GOPROXY_OLD)
lint:vet
	@echo "Running golangci-lint"
	@hash golangci-lint > /dev/null 2>&1 || $(install_golangci_lint)
	@golangci-lint run
upgrade:
	@echo "Upgrading..."
	@go env -w GOPROXY=$(GOPROXY)
	@$(TIDY_CMD)
	@go get -d -u -v ./...
	@$(TIDY_CMD)
	@go env -w GOPROXY=$(GOPROXY_OLD)
	@echo "Upgrade complete."
remote_url=$(shell git config --local --get remote.origin.url)
git_config=(git config user.name "charles" && git config user.email "-")
git-push:
	@echo "Initializing commit with remote url:$(remote_url)"
	@rm -rf .git && git init && git checkout --orphan master
	@git add .
	@$(git_config)
	@git commit -am "Initial commit"
	@git remote add origin "$(remote_url)"
	@git push -f origin master
	@git branch --set-upstream-to=origin/master master
	@git pull
git-pull:
	@rm -rf ./.git && git init
	@git remote add origin "$(remote_url)"
	@git checkout --orphan latest_branch
	@git add -A
	@$(git_config)
	@git commit -am "Initial commit" && git fetch
	@git checkout -b master origin/master
	@git branch -D latest_branch
	@git pull