workspace "CloudChamber" "C4 model for the CloudChamber service platform (Ardan-style layered Go services)." {

    !identifiers hierarchical

    model {
        !impliedRelationships false

        operator = person "Operator" "Runs and develops the platform; uses admin tools and APIs."
        endUser = person "End user" "Uses product APIs through clients."

        cloudchamber = softwareSystem "CloudChamber" "Go microservices: main API (CloudChamber), auth, metrics, and PostgreSQL." {
            !docs docs
            !adrs decisions

            group "Control plane & clients" {
                adminUI = container "Admin UI" "Optional staff web app (Vue)." "Vue / Vite SPA" "Web Browser"
            }
            group "Application services" {
                apiService = container "API service" "Primary HTTP API and domain routes. Starts and queries pipeline workflows." "Go" {
                    auditapp = component "auditapp" "Audit subdomain (app/domain/auditapp, business/domain/auditbus)." "Go"
                    productapp = component "productapp" "Product subdomain (app/domain/productapp, business/domain/productbus)." "Go"
                    userapp = component "userapp" "User subdomain (app/domain/userapp, business/domain/userbus)." "Go"
                    vproductapp = component "vproductapp" "VProduct read-model subdomain (app/domain/vproductapp, business/domain/vproductbus)." "Go"
                    projectapp = component "projectapp" "Project subdomain (app/domain/projectapp, business/domain/projectbus). One owner, many files and jobs. A file is recorded only in a project that owner owns." "Go"
                    pipelineapp = component "pipelineapp" "Starts a conversion from a file id. The job copies the owner and the project from that file; it does not take a project from the client." "Go / Temporal client"
                    tusdapp = component "tusdapp" "Internal tusd hook subdomain (app/domain/tusdapp). pre-create authenticates the caller and checks they own the project in the upload metadata. pre-finish records the upload and the artifact, in that project, before tusd answers. pre-terminate refuses once the artifact exists. Download is disabled." "Go"
                }
                authAPI = container "Auth service" "JWT issue and validation." "Go" {
                    authapp = component "authapp" "Authentication/token HTTP subdomain (app/domain/authapp)." "Go"
                    grpcauthapp = component "grpcauthapp" "gRPC token validation subdomain for peer services (app/domain/grpcauthapp)." "Go"
                    userbus = component "userbus" "Identity/user records subdomain used for authentication (business/domain/userbus)." "Go"
                }
                uploadService = container "Upload service (tusd)" "tusd server for resumable video/audio uploads. Stores bytes in artifact storage. Hooks on the API service check the project, record the artifact, and refuse download. The browser never reads bytes from tusd." "tusd (Go)" "Service"
            }
            group "Pipeline orchestration (Temporal)" {
                temporalServer = container "Temporal Service" "Durable workflow orchestration cluster/frontend for workflows, task queues, retries, and visibility (dev: auto-setup)." "Temporal" "Service"
                temporalUI = container "Temporal UI" "Web interface for workflow visibility and operations." "Temporal UI" "Web Browser"
            }
            group "Pipeline execution" {
                workflowWorker = container "Workflow worker" "Runs ProcessSourceWorkflow on the workflow task queue." "Go / Temporal worker" "Service"
                mediaWorker = container "Media worker" "Source, audio, and key-frame activities on the media task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
                asrWorker = container "ASR worker" "Transcription activities on the ASR task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
                renderWorker = container "Render worker" "Compose and export activities on the render task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
            }
            group "Artifact & data stores" {
                postgres = container "PostgreSQL" "Relational data for domains and migrations." "PostgreSQL" "Database"
                temporalPostgres = container "Temporal PostgreSQL" "Persistence and visibility storage for Temporal (physical isolation in dev)." "PostgreSQL" "Database"
                artifactStore = container "Artifact storage (S3-compatible)" "S3-compatible object storage for source media, audio, transcripts, key frames, and exported documents. The durable name is the file id. Each file belongs to one owner and one project. Dev: SeaweedFS." "SeaweedFS (S3-compatible)" "Database"
            }
            group "Observability" {
                metricsAPI = container "Metrics service" "Metrics scrape and publish sidecar patterns." "Go"
            }
        }

        // C1 — system context
        endUser -> cloudchamber "Uses product APIs on" "HTTPS"
        operator -> cloudchamber "Operates and administers" "HTTPS"

        // C2 — containers (container view only; no component endpoints)
        endUser -> cloudchamber.authAPI "Authenticates through" "HTTPS/JSON"
        endUser -> cloudchamber.apiService "Calls business APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.uploadService "Uploads recordings to, resumably (tus)" "HTTPS"
        operator -> cloudchamber.adminUI "Opens admin UI in" "HTTPS"
        cloudchamber.adminUI -> cloudchamber.authAPI "Authenticates through" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService "Makes API requests to" "HTTPS/JSON"
        cloudchamber.apiService -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService -> cloudchamber.authAPI "Validates JWTs through" "gRPC"
        cloudchamber.authAPI -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.metricsAPI -> cloudchamber.apiService "Scrapes metrics from" "HTTP"
        operator -> cloudchamber.temporalUI "Inspects workflows in" "HTTPS"
        cloudchamber.temporalUI -> cloudchamber.temporalServer "Connects to" "gRPC"
        cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService -> cloudchamber.temporalServer "Starts and queries CloudChamber pipeline workflows via" "gRPC"
        cloudchamber.apiService -> cloudchamber.artifactStore "Signs a short-lived read URL, and deletes an object the caller owns"
        endUser -> cloudchamber.artifactStore "Reads an owned file through a short-lived URL" "S3 API"
        cloudchamber.uploadService -> cloudchamber.artifactStore "Stores upload data in" "S3 API"
        cloudchamber.uploadService -> cloudchamber.apiService "Calls hooks (pre-create, pre-finish, pre-terminate, post-terminate)" "HTTPS/JSON"
        cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls workflow task queue and executes workflows via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls media task queue and executes activities via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.artifactStore "Reads the source and writes audio and key frames in the same project"
        cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls ASR task queue and executes activities via" "gRPC"
        cloudchamber.asrWorker -> cloudchamber.artifactStore "Reads audio and writes the transcript in the same project"
        cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls render task queue and executes activities via" "gRPC"
        cloudchamber.renderWorker -> cloudchamber.artifactStore "Reads the inputs and writes the document in the same project"

        // C3 — API service (subdomains aligned to current packages)
        endUser -> cloudchamber.apiService.userapp "Uses user APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.productapp "Uses product APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.projectapp "Creates and opens projects on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.pipelineapp "Starts a conversion from a file id on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.auditapp "Uses audit APIs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.userapp "Manages users via" "HTTPS/JSON"
        cloudchamber.apiService.auditapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.productapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.userapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.vproductapp -> cloudchamber.postgres "Reads from" "SQL/TCP"
        cloudchamber.apiService.projectapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.postgres "Copies the source file's owner and project onto the job" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.temporalServer "Starts and queries workflows via" "gRPC"
        cloudchamber.apiService -> cloudchamber.authAPI.grpcauthapp "Validates tokens via (service-to-service)" "gRPC"

        // C3 — Auth service (subdomains aligned to current packages)
        endUser -> cloudchamber.authAPI.authapp "Authenticates through" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.authAPI.authapp "Authenticates through" "HTTPS/JSON"
        cloudchamber.authAPI.authapp -> cloudchamber.authAPI.userbus "Loads identity through"
        cloudchamber.authAPI.grpcauthapp -> cloudchamber.authAPI.userbus "Validates identity through"
        cloudchamber.authAPI.userbus -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"

        development = deploymentEnvironment "Development" {
            deploymentNode "Developer machine" "Browser, kubectl, and Kind cluster access." "Linux workstation" "1" {
                deploymentNode "Local web tier" "Admin UI via Vite; API via localhost port-forwards or Kind hostPort." "Web Browser" "1" {
                    containerInstance cloudchamber.adminUI
                }
                deploymentNode "Kind cluster (cloudchamber-dev)" "KIND config under zarf/k8s/dev/kind-config.yaml." "kind.x-k8s.io" "1" {
                    deploymentNode "Namespace cloudchamber-system" "Kustomize manifests under zarf/k8s/dev/*." "Kubernetes namespace" "1" {
                        deploymentNode "Application services" "Core API, auth, and metrics workloads." "Workload group" "1" {
                            deploymentNode "Workload: CloudChamber" "CloudChamber API + metrics sidecar pattern." "Deployment" "1" {
                                containerInstance cloudchamber.apiService
                                containerInstance cloudchamber.metricsAPI
                            }
                            deploymentNode "Workload: auth" "Auth service Deployment." "Deployment" "1" {
                                containerInstance cloudchamber.authAPI
                            }
                            deploymentNode "Workload: tusd" "tusd resumable upload server (hooks to cloudchamber API, stores to SeaweedFS S3)." "Deployment" "1" {
                                containerInstance cloudchamber.uploadService
                            }
                        }
                        deploymentNode "Pipeline orchestration" "Temporal control plane and visibility (dev)." "Workload group" "1" {
                            deploymentNode "Workload: temporal-server" "Temporal server (auto-setup, dev)." "Deployment" "1" {
                                containerInstance cloudchamber.temporalServer
                            }
                            deploymentNode "Workload: temporal-ui" "Temporal UI." "Deployment" "1" {
                                containerInstance cloudchamber.temporalUI
                            }
                        }
                        deploymentNode "Artifact & data stores" "Application, Temporal, and artifact persistence." "Workload group" "1" {
                            deploymentNode "Workload: database" "PostgreSQL StatefulSet or external DB." "StatefulSet" "1" {
                                containerInstance cloudchamber.postgres
                            }
                            deploymentNode "Workload: temporal-database" "Temporal PostgreSQL StatefulSet (physical isolation, dev)." "StatefulSet" "1" {
                                containerInstance cloudchamber.temporalPostgres
                            }
                            deploymentNode "Workload: seaweedfs" "SeaweedFS S3-compatible artifact storage (dev, weed mini)." "Deployment" "1" {
                                containerInstance cloudchamber.artifactStore
                            }
                        }
                        deploymentNode "Pipeline workers" "One deployment for each task queue. Dev activities are simulated stand-ins." "Workload group" "1" {
                            deploymentNode "Workload: workflow-worker" "Workflow worker." "Deployment" "1" {
                                containerInstance cloudchamber.workflowWorker
                            }
                            deploymentNode "Workload: media-worker" "Media worker." "Deployment" "1" {
                                containerInstance cloudchamber.mediaWorker
                            }
                            deploymentNode "Workload: asr-worker" "ASR worker." "Deployment" "1" {
                                containerInstance cloudchamber.asrWorker
                            }
                            deploymentNode "Workload: render-worker" "Render worker." "Deployment" "1" {
                                containerInstance cloudchamber.renderWorker
                            }
                        }
                    }
                }
            }
        }

        production = deploymentEnvironment "Production" {
            deploymentNode "Client access" "End users and operators access the platform over HTTPS." "Internet / corporate network" "1" {
                deploymentNode "Ingress / load balancing" "TLS termination, routing, WAF/rate-limiting as needed." "Ingress / LB" "1" {
                    infrastructureNode "HTTPS entrypoint" "Routes to admin UI, API, and auth endpoints." "Load balancer / ingress"
                }
            }

            deploymentNode "Kubernetes / cloud platform (conceptual)" "Vendor-neutral production runtime; workloads scaled and operated per profile." "Managed platform" "1" {
                deploymentNode "Application tier" "Stateless/control-plane services." "Workload group" "1" {
                    deploymentNode "admin-ui" "Static/admin UI delivery." "Deployment / CDN" "1" {
                        containerInstance cloudchamber.adminUI
                    }
                    deploymentNode "api-service" "Primary API. Temporal client for pipeline submission and query." "Deployment (HA)" 1 {
                        containerInstance cloudchamber.apiService
                    }
                    deploymentNode "auth-service" "Token issuance/validation." "Deployment (HA)" 1 {
                        containerInstance cloudchamber.authAPI
                    }
                    deploymentNode "upload-service (tusd)" "tusd resumable upload server; hooks to the API service, stores to S3-compatible artifact storage." "Deployment (HA)" 1 {
                        containerInstance cloudchamber.uploadService
                    }
                    deploymentNode "metrics" "Metrics scrape/publish." "Deployment / sidecar" "1" {
                        containerInstance cloudchamber.metricsAPI
                    }
                }

                deploymentNode "Pipeline orchestration (Temporal)" "Managed or self-hosted HA Temporal cluster." "Workload group" "1" {
                    deploymentNode "temporal-service" "Temporal frontend/history/matching/worker services (HA)." "Temporal cluster" 1 {
                        containerInstance cloudchamber.temporalServer
                    }
                    deploymentNode "temporal-ui" "Workflow visibility (restricted access)." "Deployment" "1" {
                        containerInstance cloudchamber.temporalUI
                    }
                }

                deploymentNode "Pipeline execution (split by capability)" "Independent worker pools scaled by queue/profile; idempotent activities, heartbeats/timeouts." "Workload group" "1" {
                    deploymentNode "workflow-workers" "Poll cloudchamber-workflow queue." "Deployment (HPA)" 1 {
                        containerInstance cloudchamber.workflowWorker
                    }
                    deploymentNode "media-workers" "CPU/I/O media activities (ffmpeg/extract)." "Deployment (HPA)" 1 {
                        containerInstance cloudchamber.mediaWorker
                    }
                    deploymentNode "asr-workers" "Transcription activities (CPU/GPU/model-backed)." "Deployment (HPA)" 1 {
                        containerInstance cloudchamber.asrWorker
                    }
                    deploymentNode "render-workers" "Compose/export (Markdown/Typst PDF)." "Deployment (HPA)" 1 {
                        containerInstance cloudchamber.renderWorker
                    }
                }

                deploymentNode "Artifact & data stores" "Managed/persistent stores; workflows pass refs/IDs only." "Workload group" "1" {
                    deploymentNode "application-db" "Primary relational database (HA/backups as required)." "Managed PostgreSQL / HA" "1" {
                        containerInstance cloudchamber.postgres
                    }
                    deploymentNode "temporal-persistence" "Temporal persistence/visibility database (operated for HA)." "Managed PostgreSQL / HA" "1" {
                        containerInstance cloudchamber.temporalPostgres
                    }
                    deploymentNode "artifact-storage" "S3-compatible object storage (e.g. SeaweedFS production topology or managed S3) for source/audio/transcripts/frames/exports." "S3-compatible object store" "1" {
                        containerInstance cloudchamber.artifactStore
                    }
                }

                deploymentNode "Observability & operations" "Logs, metrics, traces, alerts (conceptual)." "Platform services" "1" {
                    infrastructureNode "Telemetry stack" "OTEL/metrics/logs/traces, dashboards, alerting." "Observability"
                }
            }
        }
    }

    views {
        systemContext cloudchamber "SystemContext" {
            title "System Context View: CloudChamber"
            description "People and the CloudChamber software system."
            include operator endUser cloudchamber
            default
            autoLayout tb
        }

        container cloudchamber "Containers" {
            title "Container View: CloudChamber"
            description "Control plane, application services, Temporal orchestration, split-ready pipeline workers, artifact/data stores, and observability."
            include *
            autoLayout tb
        }

        dynamic cloudchamber "UploadTusHooksDynamic" {
            title "Dynamic View: Upload recording"
            description "Resumable upload of a user-provided recording."

            endUser -> cloudchamber.uploadService "Creates an upload for a project"
            cloudchamber.uploadService -> cloudchamber.apiService "pre-create: authenticates the caller and checks they own the project"
            cloudchamber.uploadService -> endUser "Returns the upload URL"
            endUser -> cloudchamber.uploadService "Sends the recording data"
            cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording data"
            cloudchamber.uploadService -> cloudchamber.apiService "pre-finish: reports completion before the client is answered"
            cloudchamber.apiService -> cloudchamber.artifactStore "Verifies the object is present"
            cloudchamber.apiService -> cloudchamber.postgres "Records the upload and the artifact in that project"
            cloudchamber.uploadService -> endUser "Confirms the upload is complete"

            autoLayout tb
        }

        dynamic cloudchamber "WatchArtifactDynamic" {
            title "Dynamic View: Watch a recording"
            description "The user opens a file they own. The file id is the durable name. The API checks the owner and signs a short-lived URL. Object storage serves the bytes."

            endUser -> cloudchamber.apiService "Asks for a short-lived URL for a file id"
            cloudchamber.apiService -> cloudchamber.postgres "Loads the artifact and checks the owner"
            cloudchamber.apiService -> cloudchamber.artifactStore "Signs a GetObject URL for the browser"
            cloudchamber.apiService -> endUser "Returns the URL and when it expires"
            endUser -> cloudchamber.artifactStore "Reads the recording, including range requests"

            autoLayout tb
        }

        component cloudchamber.apiService "APIServiceC3" {
            title "Component View: API service"
            description "Subdomains inside the API service. projectapp owns projects. pipelineapp copies a file's project onto a job. tusdapp records the source file in that project."
            include *
            include *?
            include cloudchamber.temporalServer
            include cloudchamber.artifactStore
            include cloudchamber.postgres
            include cloudchamber.uploadService
            autoLayout tb
        }

        component cloudchamber.authAPI "AuthServiceC3" {
            title "Component View: Auth service"
            description "Subdomains inside the Auth service, aligned to current packages: authapp, grpcauthapp, and userbus."
            include *
            include *?
            autoLayout tb
        }

        deployment * development "DeploymentDevKind" {
            title "Deployment view: Development (Kind)"
            description "Local Kind layout for cloudchamber-dev (namespace cloudchamber-system), grouped by application, orchestration, artifact/data stores, and workers."
            include *
            autoLayout tb
        }

        deployment * production "DeploymentProduction" {
            title "Deployment view: Production (conceptual)"
            description "Vendor-neutral production-like topology: HA API/Auth/Temporal, split capability worker pools, and S3-compatible artifact storage (refs/IDs only)."
            include *
            autoLayout tb
        }

        theme themes/default/theme.json

        styles {
            element "Decision:Accepted" {
                color #ffffff
                background #5cb85c
            }
            element "Decision:Proposed" {
                color #ffffff
                background #f0ad4e
            }
            element "Decision:Superseded" {
                color #ffffff
                background #999999
            }
        }
    }

    configuration {
        scope softwaresystem
    }

}
