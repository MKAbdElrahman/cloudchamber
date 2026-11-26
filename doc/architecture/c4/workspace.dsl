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
                adminUI = container "Admin UI" "Optional staff web app (Vue)." "Vue / Vite SPA" "Web Browser" {
                    tags "Product"
                }
            }
            group "Application services" {
                apiService = container "API service" "Primary HTTP API and domain routes. Starts pipeline workflows. While a job is open, reads the Activity Type the open run has reached and does not store it." "Go" {
                    tags "Product"
                    auditapp = component "auditapp" "Records account changes for an administrator to review." "Go"
                    userapp = component "userapp" "Creates and manages accounts." "Go"
                    projectapp = component "projectapp" "A person owns many projects. Each file and job belongs to one project." "Go"
                    pipelinejobapp = component "pipelinejobapp" "Starts a run from one file. Stores the run id when the start is accepted. While the job is open, reads the Activity Type the open run has reached and does not store it. The workers store the file ids and the closed outcome." "Go / Temporal client"
                    uploadapp = component "uploadapp" "Checks the project on an upload, records the file when the upload finishes, and refuses to remove a finished file." "Go"
                }
                authAPI = container "Auth service" "JWT issue and validation." "Go" {
                    tags "Product"
                    authapp = component "authapp" "Issues a sign-in token and checks it." "Go"
                    grpcauthapp = component "grpcauthapp" "Checks a token for another service." "Go"
                    userbus = component "userbus" "The account record used to sign in." "Go"
                }
                uploadService = container "Upload service" "Accepts a recording and stores it." "tusd" "Service" {
                    tags "Product"
                }
            }
            group "Pipeline orchestration (Temporal)" {
                temporalServer = container "Temporal Service" "Frontend, history, matching, and the internal worker service. Dev runs them in one auto-setup process in an isolated Kubernetes namespace. History and visibility are in Temporal PostgreSQL." "Temporal" "Service" {
                    tags "Product"
                }
                temporalUI = container "Temporal UI" "Web interface for workflow visibility and operations." "Temporal UI" "Web Browser" {
                    tags "Product"
                }
            }
            group "Pipeline execution" {
                workflowWorker = container "Workflow worker" "Runs ProcessSourceWorkflow on the workflow task queue. It does not open the database. One worker polls the queue, and a crash waits until that worker polls again." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                mediaWorker = container "Media worker" "ResolveSource, ExtractAudio, and MarkCanceled on the media task queue. ExtractAudio reads the source object and writes 16 kHz mono 16-bit PCM WAV with ffmpeg. ResolveSource and MarkCanceled stay simulated in dev." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                framesWorker = container "Frames worker" "Key-frame activities on the frames task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                asrWorker = container "ASR worker" "Transcription activities on the ASR task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                renderWorker = container "Render worker" "Compose and export activities on the render task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
            }
            group "LLM gateway" {
                litellmGateway = container "LiteLLM gateway" "LLM proxy for pipeline transcription and document steps. Dev runs in an isolated Kubernetes namespace. Exports LLM observability to Langfuse via OTel v2 (langfuse_otel), not to Tempo." "LiteLLM Proxy" "Service" {
                    tags "LLM"
                }
                litellmPostgres = container "LiteLLM PostgreSQL" "Persistence for the LLM gateway (physical isolation in dev)." "PostgreSQL" "Database" {
                    tags "LLM"
                }
            }
            group "LLM observability" {
                langfuseWeb = container "Langfuse web" "UI and OTLP ingest for LLM generations, sessions, and cost. Dev runs in an isolated Kubernetes namespace." "Langfuse" "Web Browser" {
                    tags "LLM"
                }
                langfuseWorker = container "Langfuse worker" "Background ingestion for Langfuse." "Langfuse" "Service" {
                    tags "LLM"
                }
                langfusePostgres = container "Langfuse PostgreSQL" "Langfuse transactional data (physical isolation in dev)." "PostgreSQL" "Database" {
                    tags "LLM"
                }
                langfuseClickHouse = container "Langfuse ClickHouse" "Langfuse analytics and observation storage." "ClickHouse" "Database" {
                    tags "LLM"
                }
                langfuseBlobStore = container "Langfuse blob storage" "S3-compatible media and event objects for Langfuse (dedicated SeaweedFS in dev)." "S3-compatible" "Database" {
                    tags "LLM"
                }
            }
            group "Artifact & data stores" {
                postgres = container "PostgreSQL" "Relational data for domains and migrations." "PostgreSQL" "Database" {
                    tags "Product"
                }
                temporalPostgres = container "Temporal PostgreSQL" "Persistence and visibility storage for Temporal (physical isolation in dev)." "PostgreSQL" "Database"
                artifactStore = container "Artifact storage" "Object bytes for recordings and pipeline outputs. Project membership is the artifact row in PostgreSQL." "S3-compatible" "Database" {
                    tags "Product"
                }
            }
            group "Observability" {
                metricsAPI = container "Metrics service" "Metrics scrape and publish sidecar patterns." "Go" {
                    tags "Observability"
                }
                tempo = container "Tempo" "Distributed trace storage for Go services and pipeline workers (OTLP). Dev runs in an isolated Kubernetes namespace." "Grafana Tempo" "Database" {
                    tags "Observability"
                }
                grafana = container "Grafana" "Explores traces, metrics, and logs in dev." "Grafana" "Web Browser" {
                    tags "Observability"
                }
                loki = container "Loki" "Log aggregation for dev workloads (scraped via Alloy)." "Grafana Loki" "Database" {
                    tags "Observability"
                }
                prometheus = container "Prometheus" "Metrics scrape store; scrapes the API metrics endpoint cross-namespace in dev." "Prometheus" "Database" {
                    tags "Observability"
                }
                logCollector = container "Alloy" "Collects pod logs cluster-wide and pushes to Loki." "Grafana Alloy" "Service" {
                    tags "Observability"
                }
            }
        }

        // C1 — system context
        endUser -> cloudchamber "Uses product APIs on" "HTTPS"
        operator -> cloudchamber "Operates and administers" "HTTPS"

        // C2 — containers (container view only; no component endpoints)
        endUser -> cloudchamber.authAPI "Authenticates through" "HTTPS/JSON"
        endUser -> cloudchamber.apiService "Calls business APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.uploadService "Uploads a recording" "HTTPS"
        operator -> cloudchamber.adminUI "Opens admin UI in" "HTTPS"
        cloudchamber.adminUI -> cloudchamber.authAPI "Authenticates through" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService "Makes API requests to" "HTTPS/JSON"
        cloudchamber.apiService -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService -> cloudchamber.authAPI "Validates JWTs through" "gRPC"
        cloudchamber.apiService -> cloudchamber.authAPI "Checks the caller through" "HTTPS/JSON"
        cloudchamber.authAPI -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.metricsAPI -> cloudchamber.apiService "Reads expvar from" "HTTP" "TelemetryInternal"
        operator -> cloudchamber.grafana "Explores platform telemetry in" "HTTPS" "OperatorPlatform"
        cloudchamber.grafana -> cloudchamber.tempo "Queries traces in" "HTTP" "TelemetryInternal"
        cloudchamber.grafana -> cloudchamber.prometheus "Queries metrics in" "HTTP" "TelemetryInternal"
        cloudchamber.grafana -> cloudchamber.loki "Queries logs in" "HTTP" "TelemetryInternal"
        cloudchamber.logCollector -> cloudchamber.loki "Pushes pod logs via" "HTTP" "TelemetryInternal"
        cloudchamber.prometheus -> cloudchamber.apiService "Scrapes metrics from" "HTTP" "TelemetryInternal"
        cloudchamber.prometheus -> cloudchamber.metricsAPI "Scrapes expvar path on" "HTTP" "TelemetryInternal"
        cloudchamber.apiService -> cloudchamber.tempo "Exports service traces via" "OTLP/gRPC" "TelemetryHub"
        cloudchamber.workflowWorker -> cloudchamber.tempo "Exports worker traces via" "OTLP/gRPC" "TelemetryExport"
        cloudchamber.mediaWorker -> cloudchamber.tempo "Exports worker traces via" "OTLP/gRPC" "TelemetryExport"
        cloudchamber.framesWorker -> cloudchamber.tempo "Exports worker traces via" "OTLP/gRPC" "TelemetryExport"
        cloudchamber.asrWorker -> cloudchamber.tempo "Exports worker traces via" "OTLP/gRPC" "TelemetryExport"
        cloudchamber.renderWorker -> cloudchamber.tempo "Exports worker traces via" "OTLP/gRPC" "TelemetryExport"
        operator -> cloudchamber.temporalUI "Inspects workflows in" "HTTPS"
        cloudchamber.temporalUI -> cloudchamber.temporalServer "Connects to" "gRPC"
        cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Reads from and writes to" "SQL/TCP" "PlatformData"
        cloudchamber.apiService -> cloudchamber.temporalServer "Starts and cancels workflows, and reads the Activity Type the open run has reached via" "gRPC" "WorkflowClient"
        cloudchamber.apiService -> cloudchamber.artifactStore "Signs a short-lived read URL, and deletes an object the caller owns"
        endUser -> cloudchamber.artifactStore "Reads an owned file through a short-lived URL" "S3 API"
        cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording" "S3 API"
        cloudchamber.uploadService -> cloudchamber.apiService "Asks the API to check and record the file" "HTTPS/JSON"
        cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls workflow task queue and executes workflows via" "gRPC" "WorkflowPoll"
        cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls media task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.mediaWorker -> cloudchamber.postgres "Loads the source file and registers the audio file in its project" "SQL/TCP" "PipelineData"
        cloudchamber.mediaWorker -> cloudchamber.artifactStore "Checks the source object, reads it, and writes the audio object" "S3 API" "PipelineData"
        cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls frames task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.framesWorker -> cloudchamber.postgres "Registers a key frame in the source file's project" "SQL/TCP" "PipelineData"
        cloudchamber.framesWorker -> cloudchamber.artifactStore "Writes a key-frame object" "S3 API" "PipelineData"
        cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls ASR task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript in the source file's project" "SQL/TCP" "PipelineData"
        cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object" "S3 API" "PipelineData"
        cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls render task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document in the source file's project" "SQL/TCP" "PipelineData"
        cloudchamber.renderWorker -> cloudchamber.artifactStore "Writes the document object" "S3 API" "PipelineData"
        operator -> cloudchamber.litellmGateway "Operates" "HTTPS" "OperatorPlatform"
        cloudchamber.litellmGateway -> cloudchamber.litellmPostgres "Reads from and writes to" "SQL/TCP" "LLMDataPlane"
        cloudchamber.litellmGateway -> cloudchamber.langfuseWeb "Exports LLM spans via" "OTLP/HTTP" "LLMExport"
        cloudchamber.asrWorker -> cloudchamber.litellmGateway "Calls transcription and document models through" "HTTP" "LLMCall"
        cloudchamber.renderWorker -> cloudchamber.litellmGateway "Calls document models through" "HTTP" "LLMCall"
        operator -> cloudchamber.langfuseWeb "Inspects LLM traces in" "HTTPS" "OperatorPlatform"
        cloudchamber.langfuseWeb -> cloudchamber.langfusePostgres "Reads from and writes to" "SQL/TCP" "LLMDataPlane"
        cloudchamber.langfuseWorker -> cloudchamber.langfusePostgres "Reads from and writes to" "SQL/TCP" "LLMDataPlane"
        cloudchamber.langfuseWeb -> cloudchamber.langfuseClickHouse "Reads from and writes to" "TCP" "LLMDataPlane"
        cloudchamber.langfuseWorker -> cloudchamber.langfuseClickHouse "Reads from and writes to" "TCP" "LLMDataPlane"
        cloudchamber.langfuseWeb -> cloudchamber.langfuseBlobStore "Reads from and writes objects" "S3 API" "LLMDataPlane"
        cloudchamber.langfuseWorker -> cloudchamber.langfuseBlobStore "Reads from and writes objects" "S3 API" "LLMDataPlane"

        // C3 — API service (subdomains aligned to current packages)
        endUser -> cloudchamber.apiService.userapp "Uses user APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.projectapp "Creates and opens projects on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.pipelinejobapp "Starts a conversion from a file id on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.auditapp "Uses audit APIs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.userapp "Manages users via" "HTTPS/JSON"
        cloudchamber.apiService.auditapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.userapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.projectapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.pipelinejobapp -> cloudchamber.postgres "Stores the job in the source file's project, the run id, and a start failure" "SQL/TCP"
        cloudchamber.apiService.pipelinejobapp -> cloudchamber.temporalServer "Starts and cancels workflows, and reads the Activity Type the open run has reached via" "gRPC"
        cloudchamber.uploadService -> cloudchamber.apiService.uploadapp "Asks the API to check and record the file" "HTTPS/JSON"
        cloudchamber.apiService.uploadapp -> cloudchamber.authAPI.grpcauthapp "Checks the caller" "gRPC"
        cloudchamber.apiService.uploadapp -> cloudchamber.artifactStore "Confirms the uploaded object is stored"
        cloudchamber.apiService.uploadapp -> cloudchamber.postgres "Records the upload and the file" "SQL/TCP"
        cloudchamber.apiService -> cloudchamber.authAPI.grpcauthapp "Validates tokens via (service-to-service)" "gRPC"

        // C3 — Auth service (subdomains aligned to current packages)
        endUser -> cloudchamber.authAPI.authapp "Authenticates through" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.authAPI.authapp "Authenticates through" "HTTPS/JSON"
        cloudchamber.authAPI.authapp -> cloudchamber.authAPI.userbus "Loads identity through"
        cloudchamber.authAPI.grpcauthapp -> cloudchamber.authAPI.userbus "Validates identity through"
        cloudchamber.authAPI.userbus -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"

        development = deploymentEnvironment "Development" {
            deploymentNode "Developer machine" "Your machine." "Linux workstation" "1" {
                deploymentNode "Local web tier" "The browser." "Web Browser" "1" {
                    containerInstance cloudchamber.adminUI
                }
                deploymentNode "Kind cluster (cloudchamber-dev)" "The local cluster." "kind.x-k8s.io" "1" {
                    deploymentNode "Namespace cloudchamber-system" "Dev namespace." "Kubernetes namespace" "1" {
                        tags "DeploymentNsProduct"
                        deploymentNode "Application services" "API, auth, and upload." "Workload group" "1" {
                            deploymentNode "Workload: CloudChamber" "API and metrics." "Deployment" "1" {
                                containerInstance cloudchamber.apiService
                                containerInstance cloudchamber.metricsAPI
                            }
                            deploymentNode "Workload: auth" "Sign-in." "Deployment" "1" {
                                containerInstance cloudchamber.authAPI
                            }
                            deploymentNode "Workload: tusd" "Uploads." "Deployment" "1" {
                                containerInstance cloudchamber.uploadService
                            }
                        }
                        deploymentNode "Artifact & data stores" "Databases and files." "Workload group" "1" {
                            deploymentNode "Workload: database" "Application data." "StatefulSet" "1" {
                                containerInstance cloudchamber.postgres
                            }
                            deploymentNode "Workload: seaweedfs" "Files." "Deployment" "1" {
                                containerInstance cloudchamber.artifactStore
                            }
                        }
                        deploymentNode "Pipeline workers" "One worker for each step. A crash waits until that worker polls again." "Workload group" "1" {
                            deploymentNode "Workload: workflow-worker" "Workflow." "Deployment" "1" {
                                containerInstance cloudchamber.workflowWorker
                            }
                            deploymentNode "Workload: media-worker" "Source, audio, and a cancel between steps." "Deployment" "1" {
                                containerInstance cloudchamber.mediaWorker
                            }
                            deploymentNode "Workload: frames-worker" "Key frames." "Deployment" "1" {
                                containerInstance cloudchamber.framesWorker
                            }
                            deploymentNode "Workload: asr-worker" "Transcript." "Deployment" "1" {
                                containerInstance cloudchamber.asrWorker
                            }
                            deploymentNode "Workload: render-worker" "Document." "Deployment" "1" {
                                containerInstance cloudchamber.renderWorker
                            }
                        }
                    }
                    deploymentNode "Namespace observability-system" "Platform telemetry namespace." "Kubernetes namespace" "1" {
                        tags "DeploymentNsObservability"
                        deploymentNode "Platform observability" "Traces, metrics, logs." "Workload group" "1" {
                            deploymentNode "Workload: tempo" "Trace store." "Deployment" "1" {
                                containerInstance cloudchamber.tempo
                            }
                            deploymentNode "Workload: prometheus" "Metrics." "Deployment" "1" {
                                containerInstance cloudchamber.prometheus
                            }
                            deploymentNode "Workload: loki" "Logs." "Deployment" "1" {
                                containerInstance cloudchamber.loki
                            }
                            deploymentNode "Workload: grafana" "Telemetry UI." "Deployment" "1" {
                                containerInstance cloudchamber.grafana
                            }
                            deploymentNode "Workload: alloy" "Log collector." "Deployment" "1" {
                                containerInstance cloudchamber.logCollector
                            }
                        }
                    }
                    deploymentNode "Namespace temporal-system" "Workflow orchestration namespace." "Kubernetes namespace" "1" {
                        tags "DeploymentNsTemporal"
                        deploymentNode "Pipeline orchestration" "Workflow control." "Workload group" "1" {
                            deploymentNode "Workload: temporal-database" "Workflow data." "StatefulSet" "1" {
                                containerInstance cloudchamber.temporalPostgres
                            }
                            deploymentNode "Workload: temporal-server" "Workflow server." "Deployment" "1" {
                                containerInstance cloudchamber.temporalServer
                            }
                            deploymentNode "Workload: temporal-ui" "Workflow view." "Deployment" "1" {
                                containerInstance cloudchamber.temporalUI
                            }
                        }
                    }
                    deploymentNode "Namespace litellm-system" "LLM gateway namespace." "Kubernetes namespace" "1" {
                        tags "DeploymentNsLitellm"
                        deploymentNode "LLM gateway" "LiteLLM proxy." "Workload group" "1" {
                            deploymentNode "Workload: litellm-database" "Gateway data." "StatefulSet" "1" {
                                containerInstance cloudchamber.litellmPostgres
                            }
                            deploymentNode "Workload: litellm" "Gateway." "Deployment" "1" {
                                containerInstance cloudchamber.litellmGateway
                            }
                        }
                    }
                    deploymentNode "Namespace langfuse-system" "LLM observability namespace." "Kubernetes namespace" "1" {
                        tags "DeploymentNsLangfuse"
                        deploymentNode "LLM observability" "Langfuse stack." "Workload group" "1" {
                            deploymentNode "Workload: langfuse-db" "Langfuse PostgreSQL." "StatefulSet" "1" {
                                containerInstance cloudchamber.langfusePostgres
                            }
                            deploymentNode "Workload: langfuse-clickhouse" "Langfuse analytics." "StatefulSet" "1" {
                                containerInstance cloudchamber.langfuseClickHouse
                            }
                            deploymentNode "Workload: langfuse-seaweedfs" "Langfuse objects." "Deployment" "1" {
                                containerInstance cloudchamber.langfuseBlobStore
                            }
                            deploymentNode "Workload: langfuse-web" "UI and OTLP." "Deployment" "1" {
                                containerInstance cloudchamber.langfuseWeb
                            }
                            deploymentNode "Workload: langfuse-worker" "Background ingest." "Deployment" "1" {
                                containerInstance cloudchamber.langfuseWorker
                            }
                        }
                    }
                }
            }
        }

        production = deploymentEnvironment "Production" {
            deploymentNode "Client access" "People using the product." "Internet" "1" {
                deploymentNode "Ingress / load balancing" "HTTPS entry." "Ingress" "1" {
                    infrastructureNode "HTTPS entrypoint" "Routes requests." "Load balancer"
                }
            }

            deploymentNode "Kubernetes / cloud platform" "Where the services run." "Managed platform" "1" {
                deploymentNode "Namespace cloudchamber-system" "Product workloads and application data plane." "Kubernetes namespace" "1" {
                    tags "DeploymentNsProduct"
                    deploymentNode "Application tier" "API, auth, upload, and admin." "Workload group" "1" {
                        deploymentNode "admin-ui" "Admin site." "Deployment" "1" {
                            containerInstance cloudchamber.adminUI
                        }
                        deploymentNode "api-service" "API and metrics." "Deployment" "1" {
                            containerInstance cloudchamber.apiService
                            containerInstance cloudchamber.metricsAPI
                        }
                        deploymentNode "auth-service" "Sign-in." "Deployment" "1" {
                            containerInstance cloudchamber.authAPI
                        }
                        deploymentNode "upload-service" "Uploads." "Deployment" "1" {
                            containerInstance cloudchamber.uploadService
                        }
                    }
                    deploymentNode "Artifact & data stores" "Application database and artifact object store." "Workload group" "1" {
                        deploymentNode "application-db" "Application data." "PostgreSQL" "1" {
                            containerInstance cloudchamber.postgres
                        }
                        deploymentNode "artifact-storage" "Recording and pipeline output bytes." "S3-compatible" "1" {
                            containerInstance cloudchamber.artifactStore
                        }
                    }
                    deploymentNode "Pipeline workers" "One worker per task queue." "Workload group" "1" {
                        deploymentNode "workflow-worker" "ProcessSourceWorkflow." "Deployment" "1" {
                            containerInstance cloudchamber.workflowWorker
                        }
                        deploymentNode "media-worker" "Source, audio, cancel." "Deployment" "1" {
                            containerInstance cloudchamber.mediaWorker
                        }
                        deploymentNode "frames-worker" "Key frames." "Deployment" "1" {
                            containerInstance cloudchamber.framesWorker
                        }
                        deploymentNode "asr-worker" "Transcript." "Deployment" "1" {
                            containerInstance cloudchamber.asrWorker
                        }
                        deploymentNode "render-worker" "Document." "Deployment" "1" {
                            containerInstance cloudchamber.renderWorker
                        }
                    }
                }
                deploymentNode "Namespace observability-system" "Platform telemetry." "Kubernetes namespace" "1" {
                    tags "DeploymentNsObservability"
                    deploymentNode "Platform observability" "Traces, metrics, logs." "Workload group" "1" {
                        deploymentNode "tempo" "Trace store." "Grafana Tempo" "1" {
                            containerInstance cloudchamber.tempo
                        }
                        deploymentNode "prometheus" "Metrics." "Prometheus" "1" {
                            containerInstance cloudchamber.prometheus
                        }
                        deploymentNode "loki" "Logs." "Grafana Loki" "1" {
                            containerInstance cloudchamber.loki
                        }
                        deploymentNode "grafana" "Telemetry UI." "Deployment" "1" {
                            containerInstance cloudchamber.grafana
                        }
                        deploymentNode "alloy" "Log collector." "Grafana Alloy" "1" {
                            containerInstance cloudchamber.logCollector
                        }
                    }
                }
                deploymentNode "Namespace temporal-system" "Workflow orchestration." "Kubernetes namespace" "1" {
                    tags "DeploymentNsTemporal"
                    deploymentNode "Pipeline orchestration" "Temporal server and UI." "Workload group" "1" {
                        deploymentNode "temporal-persistence" "Workflow data." "PostgreSQL" "1" {
                            containerInstance cloudchamber.temporalPostgres
                        }
                        deploymentNode "temporal-service" "Workflow server." "Temporal" "1" {
                            containerInstance cloudchamber.temporalServer
                        }
                        deploymentNode "temporal-ui" "Workflow view." "Deployment" "1" {
                            containerInstance cloudchamber.temporalUI
                        }
                    }
                }
                deploymentNode "Namespace litellm-system" "LLM gateway." "Kubernetes namespace" "1" {
                    tags "DeploymentNsLitellm"
                    deploymentNode "LLM gateway" "LiteLLM proxy." "Workload group" "1" {
                        deploymentNode "litellm-persistence" "Gateway data." "PostgreSQL" "1" {
                            containerInstance cloudchamber.litellmPostgres
                        }
                        deploymentNode "litellm-gateway" "Gateway." "Deployment" "1" {
                            containerInstance cloudchamber.litellmGateway
                        }
                    }
                }
                deploymentNode "Namespace langfuse-system" "LLM observability." "Kubernetes namespace" "1" {
                    tags "DeploymentNsLangfuse"
                    deploymentNode "LLM observability" "Langfuse stack." "Workload group" "1" {
                        deploymentNode "langfuse-db" "Transactional data." "PostgreSQL" "1" {
                            containerInstance cloudchamber.langfusePostgres
                        }
                        deploymentNode "langfuse-clickhouse" "Analytics." "ClickHouse" "1" {
                            containerInstance cloudchamber.langfuseClickHouse
                        }
                        deploymentNode "langfuse-blob" "Media and events." "S3-compatible" "1" {
                            containerInstance cloudchamber.langfuseBlobStore
                        }
                        deploymentNode "langfuse-web" "UI and OTLP." "Deployment" "1" {
                            containerInstance cloudchamber.langfuseWeb
                        }
                        deploymentNode "langfuse-worker" "Background ingest." "Deployment" "1" {
                            containerInstance cloudchamber.langfuseWorker
                        }
                    }
                }
            }
        }
    }

    views {
        systemContext cloudchamber "SystemContext" {
            title "System Context View: CloudChamber"
            description "People and the CloudChamber software system."
            include operator endUser cloudchamber
            autoLayout tb
        }

        container cloudchamber "ContainersProduct" {
            title "Container View: Product (recordings, API, pipeline)"
            description "End-user and operator paths through API, auth, upload, artifact storage, Temporal, and pipeline workers. One OTLP export edge stands in for every Go service and worker sending traces to Tempo (see Observability view). Platform database instances behind Temporal are omitted for brevity."
            include operator endUser
            include "element.tag==Product"
            include cloudchamber.tempo
            exclude cloudchamber.temporalPostgres
            exclude "relationship.tag==TelemetryExport"
            exclude "relationship.tag==TelemetryInternal"
            exclude "relationship.tag==LLMExport"
            exclude "relationship.tag==LLMDataPlane"
            exclude "relationship.tag==LLMCall"
            exclude "relationship.tag==OperatorPlatform"
            exclude "relationship.tag==PlatformData"
            default
            autoLayout tb
        }

        container cloudchamber "ContainersLLM" {
            title "Container View: LLM gateway and Langfuse"
            description "LiteLLM, Langfuse data plane, and pipeline workers that call models. Go worker traces stay on Tempo (Observability view)."
            include operator
            include "element.tag==LLM"
            include cloudchamber.asrWorker cloudchamber.renderWorker
            exclude "relationship.tag==TelemetryExport"
            exclude "relationship.tag==TelemetryInternal"
            exclude "relationship.tag==TelemetryHub"
            exclude "relationship.tag==WorkflowPoll"
            exclude "relationship.tag==WorkflowClient"
            exclude "relationship.tag==PipelineData"
            exclude "relationship.tag==PlatformData"
            autoLayout tb
        }

        container cloudchamber "ContainersObservability" {
            title "Container View: Platform telemetry"
            description "Tempo, Grafana, Loki, Prometheus, Alloy, and OTLP/metrics paths from product containers. LLM observability is separate (LLM view)."
            include operator
            include "element.tag==Observability"
            include cloudchamber.apiService
            include cloudchamber.workflowWorker cloudchamber.mediaWorker cloudchamber.framesWorker cloudchamber.asrWorker cloudchamber.renderWorker
            exclude "relationship.tag==WorkflowPoll"
            exclude "relationship.tag==WorkflowClient"
            exclude "relationship.tag==PipelineData"
            exclude "relationship.tag==LLMExport"
            exclude "relationship.tag==LLMDataPlane"
            exclude "relationship.tag==LLMCall"
            exclude "relationship.tag==OperatorPlatform"
            exclude "relationship.tag==PlatformData"
            autoLayout tb
        }

        container cloudchamber "ContainersFull" {
            title "Container View: Full (review)"
            description "All containers and relationships in the model. Prefer ContainersProduct, ContainersLLM, or ContainersObservability for reading."
            include *
            autoLayout tb
        }

        dynamic cloudchamber "UploadTusHooksDynamic" {
            title "Dynamic View: Upload recording"
            description "Resumable upload of a user-provided recording."

            endUser -> cloudchamber.uploadService "Starts an upload in a project"
            cloudchamber.uploadService -> cloudchamber.apiService "Checks the person owns the project"
            cloudchamber.uploadService -> endUser "Returns an address for the recording"
            endUser -> cloudchamber.uploadService "Sends the recording"
            cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording"
            cloudchamber.uploadService -> cloudchamber.apiService "Asks the API to record the file"
            cloudchamber.apiService -> cloudchamber.artifactStore "Confirms the recording is stored"
            cloudchamber.apiService -> cloudchamber.postgres "Records the file in the project"
            cloudchamber.uploadService -> endUser "Confirms the upload"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineProcessDynamic" {
            title "Dynamic View: Convert a recording"
            description "Successful run of one file, in this order. The API checks the caller, loads the source file, records the job, and starts ProcessSourceWorkflow. History records the execution and schedules the first workflow task, and that start call then returns. The API stores the run id, marks the job running, and returns the job. The workflow worker polls the first workflow task after it is queued. That poll may overlap the API response. That task schedules ResolveSource. Each later activity waits for the previous result and is scheduled by its own workflow task: ExtractAudio on media, Transcribe on ASR, ExtractKeyFrames on frames, ComposeExport on render. ResolveSource loads the source row and checks the object. Each writing step stores its object, registers it in the source file's project, and records that file id on the job. A video records a key frame after the transcript. Audio reports that step with no frame. ComposeExport records the document, then stores completed and clears the failure message and the stopped activity type. The next workflow task commands completion. Listing and opening the job are later requests, not spans in the workflow trace. A list reads the job rows and does not ask the frontend. Opening a job whose row is still open reads the Activity Type the open run has reached and does not store it. A row that is already closed is not read from the frontend again."

            endUser -> cloudchamber.apiService "Starts a conversion from one file"
            cloudchamber.apiService -> cloudchamber.authAPI "Authenticates the caller" "HTTPS/JSON"
            cloudchamber.apiService -> cloudchamber.authAPI "Authorizes the caller" "HTTPS/JSON"
            cloudchamber.apiService -> cloudchamber.postgres "Loads the source file"
            cloudchamber.apiService -> cloudchamber.postgres "Records the job with that file's owner and project"
            cloudchamber.apiService -> cloudchamber.temporalServer "Starts ProcessSourceWorkflow with the job and file ids"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the execution and the first workflow task"
            cloudchamber.apiService -> cloudchamber.postgres "Stores the run id and marks the job running"
            cloudchamber.apiService -> endUser "Returns the job for the execution just started"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the first workflow task after it is queued. This poll may overlap the API response"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Commands scheduling of ResolveSource from that same workflow task"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ResolveSource task"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Loads the source file"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Checks the source object"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ResolveSource result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ExtractAudio"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ExtractAudio task"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Reads the source object and writes 16 kHz mono PCM audio"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Registers the audio file in the source file's project"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Records the audio file id on the job"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ExtractAudio result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task"
            cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object"
            cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript in the source file's project"
            cloudchamber.asrWorker -> cloudchamber.postgres "Records the transcript file id on the job"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports the Transcribe result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ExtractKeyFrames"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls the ExtractKeyFrames task"
            cloudchamber.framesWorker -> cloudchamber.artifactStore "For a video, writes a key frame"
            cloudchamber.framesWorker -> cloudchamber.postgres "For a video, registers that frame in the source file's project"
            cloudchamber.framesWorker -> cloudchamber.postgres "For a video, records that frame id on the job"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Reports the ExtractKeyFrames result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ComposeExport"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls the ComposeExport task"
            cloudchamber.renderWorker -> cloudchamber.artifactStore "Writes the document object"
            cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document in the source file's project"
            cloudchamber.renderWorker -> cloudchamber.postgres "Records the document file id on the job"
            cloudchamber.renderWorker -> cloudchamber.postgres "Stores completed and clears the failure message and the stopped activity type"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Reports the ComposeExport result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands completion of the workflow"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the completed workflow"

            endUser -> cloudchamber.apiService "Lists jobs in the project"
            cloudchamber.apiService -> cloudchamber.postgres "Reads the job rows"

            endUser -> cloudchamber.apiService "Opens the job while it is still running"
            cloudchamber.apiService -> cloudchamber.temporalServer "Reads the Activity Type the open run has reached and does not store it"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineFailureDynamic" {
            title "Dynamic View: A conversion fails"
            description "A transient activity error is retried on the same activity, and the job stays running. A permanent activity error, or the last of five attempts, is stored by that activity before it returns. This run hits a permanent Transcribe failure after audio was stored. Transcribe stores failed, its failure message, and the stopped activity type. The workflow returns that error and closes as failed. The audio stays registered. Opening the job reads that row and does not ask the frontend."

            endUser -> cloudchamber.apiService "Starts a conversion from one file"
            cloudchamber.apiService -> cloudchamber.postgres "Loads the source file"
            cloudchamber.apiService -> cloudchamber.postgres "Records the job with that file's owner and project"
            cloudchamber.apiService -> cloudchamber.temporalServer "Starts ProcessSourceWorkflow with the job and file ids"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the execution and the first workflow task"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the workflow task and commands scheduling of ExtractAudio"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ExtractAudio task"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Reads the source object and writes 16 kHz mono PCM audio"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Registers the audio file in the source file's project"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Records the audio file id on the job"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ExtractAudio result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task"
            cloudchamber.asrWorker -> cloudchamber.postgres "Stores failed, the failure message, and the stopped activity type"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports the permanent Transcribe failure. That activity is not retried"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and returns that error"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the failed workflow. The registered audio stays"

            endUser -> cloudchamber.apiService "Opens the job"
            cloudchamber.apiService -> cloudchamber.postgres "Reads the failed job. The row already holds the outcome"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineCancelDynamic" {
            title "Dynamic View: Cancel a conversion"
            description "Cancel arrives while Transcribe is in flight, after audio was stored. The API requests cancellation and does not change the job row. The workflow waits for Transcribe to close. Transcribe stops. If it had already registered the transcript, that file id stays on the job. It stores canceled and its own name as the stopped activity type, with no failure message, then returns the cancel. The workflow does not schedule ExtractKeyFrames. The execution closes as canceled. A cancel that arrives after a step returns and before the next is scheduled is stored by MarkCanceled on the media queue, using the step already finished, and the workflow waits for that write before it returns the same cancel. Opening the job reads the row and does not ask the frontend."

            endUser -> cloudchamber.apiService "Cancels the job"
            cloudchamber.apiService -> cloudchamber.temporalServer "Requests cancellation of the execution"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the cancellation request and a workflow task"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls that task and requests cancellation of Transcribe, then waits for it to close"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Heartbeats and learns the cancel"
            cloudchamber.asrWorker -> cloudchamber.postgres "Stores canceled and the stopped activity type. The transcript is recorded only if it was already registered"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Returns the cancel after that write"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Returns a canceled error after Transcribe closes. ExtractKeyFrames is not scheduled"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the canceled workflow"

            endUser -> cloudchamber.apiService "Opens the job"
            cloudchamber.apiService -> cloudchamber.postgres "Reads the canceled job. The row already holds the outcome"

            autoLayout tb
        }

        dynamic cloudchamber "WatchArtifactDynamic" {
            title "Dynamic View: Watch a recording"
            description "A person plays a recording they own."

            endUser -> cloudchamber.apiService "Opens a recording"
            cloudchamber.apiService -> cloudchamber.postgres "Checks the person owns it"
            cloudchamber.apiService -> cloudchamber.artifactStore "Prepares a short-lived link"
            cloudchamber.apiService -> endUser "Returns the link"
            endUser -> cloudchamber.artifactStore "Plays the recording"

            autoLayout tb
        }

        component cloudchamber.apiService "APIServiceC3" {
            title "Component View: API service"
            description "Projects, accounts, jobs, and the upload check inside the API."
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
            description "Sign-in, token checks for other services, and the account record."
            include *
            include *?
            autoLayout tb
        }

        deployment * development "DeploymentDevKind" {
            title "Deployment view: Development"
            description "Local developer machine, browser tier, and Kind cluster with namespace isolation."
            include *
            autoLayout tb
        }

        deployment * development "DeploymentDevProduct" {
            title "Deployment view: Development — cloudchamber-system"
            description "Product workloads, application data, artifacts, and pipeline workers."
            include "element.tag==DeploymentNsProduct"
            autoLayout tb
        }

        deployment * development "DeploymentDevObservability" {
            title "Deployment view: Development — observability-system"
            description "Tempo, Grafana, Loki, Prometheus, and Alloy."
            include "element.tag==DeploymentNsObservability"
            autoLayout tb
        }

        deployment * development "DeploymentDevTemporal" {
            title "Deployment view: Development — temporal-system"
            description "Temporal server, UI, and persistence."
            include "element.tag==DeploymentNsTemporal"
            autoLayout tb
        }

        deployment * development "DeploymentDevLitellm" {
            title "Deployment view: Development — litellm-system"
            description "LiteLLM gateway and its database."
            include "element.tag==DeploymentNsLitellm"
            autoLayout tb
        }

        deployment * development "DeploymentDevLangfuse" {
            title "Deployment view: Development — langfuse-system"
            description "Langfuse web, worker, and data stores."
            include "element.tag==DeploymentNsLangfuse"
            autoLayout tb
        }

        deployment * production "DeploymentProduction" {
            title "Deployment view: Production"
            description "Ingress, Kubernetes platform, and namespace boundaries matching dev."
            include *
            autoLayout tb
        }

        deployment * production "DeploymentProdProduct" {
            title "Deployment view: Production — cloudchamber-system"
            description "Product tier, data stores, and pipeline workers."
            include "element.tag==DeploymentNsProduct"
            autoLayout tb
        }

        deployment * production "DeploymentProdObservability" {
            title "Deployment view: Production — observability-system"
            description "Platform telemetry stack."
            include "element.tag==DeploymentNsObservability"
            autoLayout tb
        }

        deployment * production "DeploymentProdTemporal" {
            title "Deployment view: Production — temporal-system"
            description "Temporal orchestration tier."
            include "element.tag==DeploymentNsTemporal"
            autoLayout tb
        }

        deployment * production "DeploymentProdLitellm" {
            title "Deployment view: Production — litellm-system"
            description "LLM gateway tier."
            include "element.tag==DeploymentNsLitellm"
            autoLayout tb
        }

        deployment * production "DeploymentProdLangfuse" {
            title "Deployment view: Production — langfuse-system"
            description "LLM observability tier."
            include "element.tag==DeploymentNsLangfuse"
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
