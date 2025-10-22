# PFlow 全量工单工作流演示

本仓库提供了一套基于 Go、Camunda、PostgreSQL、RabbitMQ 与 React 的工单审批示例系统，核心目标是让审批流程与业务服务解耦，方便在其他场景复用工作流引擎。整体方案包含：

- **工作流引擎**：Camunda Platform Run（通过 REST API 与外部任务模式与业务服务交互）。
- **后端服务**：Go 编写的 API，负责工单持久化、调用 Camunda 启动流程、消费外部任务以及投递 RabbitMQ 事件。
- **消息队列**：RabbitMQ，实现事件解耦，可扩展下游异步能力。
- **数据库**：PostgreSQL 持久化工单数据。
- **前端**：React + Vite 实现的审批界面。
- **部署脚本**：Docker Compose 一键启动所有依赖组件。

## 目录结构

```
backend/        Go 服务源码
frontend/       React 前端
deploy/         部署与工作流定义（BPMN）
```

## 快速开始

> **提示**：由于本环境无法联网，`go mod tidy`/`npm install` 需在可访问公网的机器执行，以拉取依赖。

1. **确保本机已安装 Docker / Docker Compose 且守护进程已启动。**

   - macOS/Windows 用户请先启动 Docker Desktop；
   - Linux 用户可执行 `sudo systemctl start docker` 或参考发行版文档启动 `dockerd`。

   若在执行 Compose 时看到 `Cannot connect to the Docker daemon` 等错误，即表示守护进程未启动，需要先按上述方式启动后再重试。

2. 克隆仓库后在根目录执行 Compose 编排。**首次使用前请确认已安装 Docker Compose V2**：

   - 推荐在 Linux 上安装官方 Compose 插件：

     ```bash
     sudo apt-get remove docker-compose  # 如曾通过 apt 安装 python2 版本需先卸载
     sudo apt-get update
     # 若未配置 Docker 官方仓库，可按下述一次性命令添加（Ubuntu/Debian）：
     # sudo apt-get install ca-certificates curl gnupg
     # sudo install -m 0755 -d /etc/apt/keyrings
     # curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
     # echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$(. /etc/os-release && echo "$ID") $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
     # sudo apt-get update
     sudo apt-get install docker-compose-plugin
     docker compose version
     ```

     以上命令会安装 `docker compose` 子命令（基于 Go 的 V2 版本），避免 `AttributeError: 'module' object has no attribute 'unique'` 等由于旧版 Python Compose 引起的报错。

     如果你的发行版仓库长期未提供 `docker-compose-plugin`（例如某些企业内网镜像站），可直接下载官方发布的二进制并手动安装：

     ```bash
     sudo mkdir -p /usr/local/lib/docker/cli-plugins
     sudo curl -L "https://github.com/docker/compose/releases/download/v2.24.7/docker-compose-$(uname -s)-$(uname -m)" \
       -o /usr/local/lib/docker/cli-plugins/docker-compose
     sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
     docker compose version
     ```

     如目标机器无法直接访问 GitHub，可先在可联网环境下载相应的 `docker-compose-<OS>-<ARCH>` 文件后，通过离线方式拷贝到上述目录，仍然保持可执行权限即可。

   - macOS/Windows 用户使用 Docker Desktop 会自带 Compose V2，无需额外安装。

   - 如仍需使用 legacy `docker-compose` 二进制，请确保是 1.29+ 版本，并安装在 Python 3 环境下。

   安装完成后，根据本地 Docker 版本选择以下任一命令启动：

   - Docker Compose **V2（`docker compose` 子命令）**：

     ```bash
     docker compose -f deploy/docker-compose.yml up --build
     ```

   - 传统的 **docker-compose 可执行文件（V1）**：

     ```bash
     docker-compose -f deploy/docker-compose.yml up --build
     ```

   若执行 `docker compose` 时提示 `unknown shorthand flag: 'f' in -f`，说明当前 Docker 客户端尚未启用 Compose V2，请改用 `docker-compose` 命令或升级 Docker 版本。

   该命令会启动以下容器：

   - `db`：PostgreSQL 15
   - `camunda`：Camunda Run，REST 端口映射为 `http://localhost:8081`
   - `rabbitmq`：消息队列，管理界面位于 `http://localhost:15672`
   - `api`：Go 工单服务，监听 `http://localhost:8080`
   - `frontend`：打包后的前端，监听 `http://localhost:5173`

3. 打开浏览器访问 [http://localhost:5173](http://localhost:5173) 体验前端，创建工单后可提交审批并在不同状态之间流转。

4. 若需要在本地调试 Go/前端代码，可参考下述“本地调试流程”小节。

### 本地调试流程

当你需要单步调试、打断点或对代码做热更新时，可将 Compose 作为依赖容器编排，业务代码在宿主机直接运行：

1. **启动依赖容器（Camunda、PostgreSQL、RabbitMQ）**

   ```bash
   docker compose -f deploy/docker-compose.yml up -d db camunda rabbitmq
   ```

   - `-d` 表示后台运行，便于在当前终端继续执行其他命令；
   - 若需查看容器日志，可使用 `docker compose logs -f camunda` 等命令。

2. **准备后端环境并运行 API**

   ```bash
   cd backend
   go mod download        # 首次或依赖变更时执行
   export DATABASE_URL="postgres://pflow:pflow@localhost:5432/pflow?sslmode=disable"
   export CAMUNDA_URL="http://localhost:8081/engine-rest"
   export CAMUNDA_PROCESS_KEY="ticket_approval"
   export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"
   export RABBITMQ_TICKET_EXCHANGE="ticket.events"
   export RABBITMQ_TICKET_QUEUE="ticket.events.queue"
   export API_HTTP_PORT=":8080"
   go run ./cmd/api
   ```

   如需断点调试，可将 `go run` 替换为 IDE/`dlv` 等工具。服务启动后会自动部署 BPMN 模型并监听 `http://localhost:8080`。

3. **运行前端 Dev Server（带实时刷新与接口代理）**

   ```bash
   cd frontend
   npm install           # 首次执行
   VITE_API_URL="http://localhost:8080" npm run dev -- --host
   ```

   - `VITE_API_URL` 会传递到 Vite 代理，确保 `/api` 请求转发到本地 Go 服务；
   - `--host` 选项便于在局域网或虚拟机下通过 IP 访问。

   Dev Server 默认监听 `http://localhost:5173`，前端代码保存后浏览器会自动刷新。

4. **停用环境**

   当调试结束后，可通过以下命令停止依赖容器并清理资源：

   ```bash
   docker compose -f deploy/docker-compose.yml down
   ```

如需在容器中调试，可直接进入对应容器执行命令，例如 `docker compose exec api sh`。

### macOS 虚拟环境（Lima）

若希望在 macOS 上完全隔离开发环境、避免直接在宿主机安装 Docker/Golang/Node.js，可以借助 [Lima](https://github.com/lima-vm/lima) 启动一台轻量级的 Linux 虚拟机。仓库在 `ops/mac-dev/lima.yaml` 中提供了开箱即用的配置，VM 内已预置 Docker、Docker Compose、Go 1.21、Node.js 18 以及常用的 CLI 依赖。

1. **安装 Lima 与（可选的）CLI 辅助工具**

   ```bash
   brew install lima jq
   ```

   Lima 使用 macOS 自带的虚拟化框架，Apple Silicon 与 Intel Mac 均可使用。

2. **启动虚拟机并挂载当前仓库**

   ```bash
   # 在仓库根目录执行
   limactl start --name pflow-dev ops/mac-dev/lima.yaml
   ```

   首次执行会下载 Ubuntu 云镜像并完成一次性初始化。初始化脚本会自动安装 Docker、Go、Node.js、pnpm 等依赖，并将当前仓库挂载到 `/mnt/lima/source/PFlow`。

3. **进入虚拟机执行开发命令**

   ```bash
   limactl shell pflow-dev
   cd /mnt/lima/source/PFlow
   docker compose -f deploy/docker-compose.yml up --build
   ```

   后续即可在 VM 中运行所有 `docker compose`、`go run`、`npm run dev` 等命令。若需要使用 VS Code Remote - SSH，可通过 `limactl ssh pflow-dev` 获取 SSH 会话。

4. **关闭或删除虚拟机**

   ```bash
   limactl stop pflow-dev          # 停止但保留 VM
   limactl delete pflow-dev        # 删除 VM（下次需重新初始化）
   ```

> **提示**：如需调整虚拟机 CPU/内存、镜像或挂载路径，可编辑 `ops/mac-dev/lima.yaml`；Apple Silicon 用户默认会自动拉取 ARM64 镜像，无需额外配置。

### 常见问题

- **`error getting credentials - err: exit status 1, out: GDBus.Error:org.freedesktop.DBus.Error.ServiceUnknown`**

  该报错表示 Docker CLI 尝试通过桌面凭据管理器存取镜像仓库凭据，但当前 Linux 服务器并未提供 `org.freedesktop.secrets` 服务。常见解决方案：

  1. **安装兼容的凭据守护进程**（如 `gnome-keyring` 或 `pass`）：

     ```bash
     sudo apt-get install gnome-keyring  # Debian/Ubuntu
     # 或者
     sudo apt-get install pass           # 轻量级 GPG 驱动
     ```

     安装后重新登录 Shell（或执行 `eval $(/usr/bin/gnome-keyring-daemon --start --components=secrets)`）再执行 `docker compose ...`。

  2. **在当前主机禁用凭据存储**，避免 Docker 请求缺失的 Secret Service。可临时覆盖 Docker 配置：

     ```bash
     mkdir -p ~/.docker
     cat > ~/.docker/config.json <<'EOF'
     {
       "auths": {}
     }
     EOF
     ```

    如原先配置存在 `"credsStore"`/`"credHelpers"` 字段，请移除后再尝试。该方式会改为在本地 `config.json` 中以明文保存凭据，请结合安全要求选用。

- **`failed to load cache key: docker/dockerfile:1 ... connect: connection refused`**

  这是 Docker BuildKit 默认尝试从 Docker Hub 拉取 `docker/dockerfile:1` 前端镜像时被防火墙拦截导致的。解决方式：

  1. **确保已经在目标主机预先导入该镜像**（例如在可联网机器执行 `docker pull docker/dockerfile:1 && docker save docker/dockerfile:1 > dockerfile.tar`，然后离线导入 `docker load < dockerfile.tar`）。
  2. 或者在离线环境暂时禁用 BuildKit，以旧版 builder 解析 Dockerfile：

     ```bash
     DOCKER_BUILDKIT=0 docker compose -f deploy/docker-compose.yml up --build
     ```

     本仓库的 Dockerfile 已移除 `# syntax=docker/dockerfile:1` 指令，禁用 BuildKit 后会完全依赖本地缓存，不再访问公网。

  如仍需使用 BuildKit，也可以通过 `docker buildx create --use` 搭建企业内私有 registry 的前端镜像镜像源，确保 `docker/dockerfile:1` 可在内网获取。

## Camunda 工作流说明

- BPMN 文件位于 `deploy/workflows/ticket-process.bpmn`，包含以下关键节点：
  - 起始事件：工单提交。
  - 用户任务 `Manager Approval`：审批人操作（可通过 Camunda Tasklist 或 API 完成）。
  - 服务任务 `Provision Service`：声明为外部任务 `ticket-processing`，由 Go 服务的外部任务 worker 轮询处理，实现自动化动作。
  - 审批通过后进入服务任务，完成后流向完成结束事件；驳回则直接结束。
- `backend/cmd/api/main.go` 在启动时自动部署 BPMN 并通过业务主键（工单 ID）与 Camunda 实例绑定。

## 后端设计亮点

- `internal/service/workflow_service.go` 封装了业务与 Camunda 交互的核心逻辑。
- `internal/worker/external_worker.go` 实现了 Camunda 外部任务 worker，可根据并发需求启动多实例扩展吞吐。
- `internal/mq/mq.go` 提供 RabbitMQ 发布/订阅接口，可按需增加消费者实现异步通知、审计等能力。
- `internal/http/server.go` 定义 REST API，前端通过 `/api/tickets` 等接口调用。
- 所有工单数据使用 `gorm` 持久化到 PostgreSQL，结构见 `internal/models/ticket.go`。

## 前端说明

- 通过 React Query 定时拉取工单列表，实现轻量实时刷新。
- 支持创建、提交、审批等核心操作，状态对应后端返回值。

## 高并发与扩展建议

- Go 服务天然支持高并发，可通过 `docker-compose`/Kubernetes 水平扩容。
- Camunda 外部任务 worker 可部署为单独微服务，并通过 RabbitMQ 或 HTTP 与主服务通信，保证工作流与业务解耦。
- RabbitMQ 事件总线使审批动作与后续处理（通知、审计、数据同步等）彻底分离，避免阻塞主流程。
- 数据库和消息队列均可替换为云托管方案以提升可靠性。

## 初始化与二次开发建议

1. 根据实际环境调整 `.env` 或 Compose 中的连接串。
2. 通过 Camunda Modeler 修改 BPMN，更新后重新部署即可。
3. 在 `frontend/` 中执行 `npm install && npm run dev` 可本地开发 UI。
4. 在 `backend/` 中执行 `go test ./...`（补充测试用例后）验证逻辑。

欢迎在此基础上扩展通知、审批人分配、表单自定义等高级功能。
