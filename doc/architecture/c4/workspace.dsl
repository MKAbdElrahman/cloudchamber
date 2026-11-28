workspace "CloudChamber" "C4 model for the CloudChamber service platform (Ardan-style layered Go services)." {

    !identifiers hierarchical

    model {
        !impliedRelationships false

        operator = person "Operator" "Runs and develops the platform; uses admin tools and APIs."
        endUser = person "End user" "Uses product APIs through clients."

        cloudchamber = softwareSystem "CloudChamber" "Turns a recording into a document." {
            !docs docs
            !adrs decisions

            group "Control plane & clients" {
                adminUI = container "Admin UI" "Optional staff web app (Vue)." "Vue / Vite SPA" "Web Browser" {
                    tags "Product"
                }
            }
            group "Application services" {
                apiService = container "API service" "HTTP API. Stores runs and drives them." "Go" {
                    tags "Product"
                    auditapp = component "auditapp" "Records account changes for an administrator to review." "Go"
                    userapp = component "userapp" "Creates and manages accounts." "Go"
                    projectapp = component "projectapp" "Projects that own files and runs." "Go"
                    runapp = component "runapp" "Creates runs and takes commands on them." "Go"
                    runcontrol = component "runcontrol" "Moves each run toward what it is asked to do." "Go / Temporal client"
                    uploadapp = component "uploadapp" "Checks and records uploaded files." "Go"
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
                temporalServer = container "Temporal Service" "Durable workflow execution." "Temporal" "Service" {
                    tags "Product"
                }
                temporalUI = container "Temporal UI" "Web interface for workflow visibility and operations." "Temporal UI" "Web Browser" {
                    tags "Product"
                }
            }
            group "Pipeline execution" {
                workflowWorker = container "Workflow worker" "Runs the workflow code of every workflow." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                mediaWorker = container "Media worker" "Resolves the source and extracts audio." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                framesWorker = container "Frames worker" "Extracts key frames." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                asrWorker = container "ASR worker" "Transcribes audio." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
                renderWorker = container "Render worker" "Composes the document." "Go / Temporal worker" "Service" {
                    tags "Product"
                }
            }
            group "LLM gateway" {
                litellmGateway = container "LiteLLM gateway" "LLM proxy for pipeline steps." "LiteLLM Proxy" "Service" {
                    tags "LLM"
                }
                litellmPostgres = container "LiteLLM PostgreSQL" "Persistence for the LLM gateway (physical isolation in dev)." "PostgreSQL" "Database" {
                    tags "LLM"
                }
            }
            group "LLM observability" {
                langfuseWeb = container "Langfuse web" "LLM traces, sessions, and cost." "Langfuse" "Web Browser" {
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
                langfuseBlobStore = container "Langfuse blob storage" "Objects for Langfuse." "S3-compatible" "Database" {
                    tags "LLM"
                }
            }
            group "Artifact & data stores" {
                postgres = container "PostgreSQL" "Relational data for domains and migrations." "PostgreSQL" "Database" {
                    tags "Product"
                }
                temporalPostgres = container "Temporal PostgreSQL" "Temporal history and visibility." "PostgreSQL" "Database"
                artifactStore = container "Artifact storage" "Recordings and pipeline outputs." "S3-compatible" "Database" {
                    tags "Product"
                }
            }
            group "Observability" {
                metricsAPI = container "Metrics service" "Metrics scrape and publish sidecar patterns." "Go" {
                    tags "Observability"
                }
                tempo = container "Tempo" "Distributed trace storage." "Grafana Tempo" "Database" {
                    tags "Observability"
                }
                grafana = container "Grafana" "Explores traces, metrics, and logs in dev." "Grafana" "Web Browser" {
                    tags "Observability"
                }
                loki = container "Loki" "Log aggregation for dev workloads (scraped via Alloy)." "Grafana Loki" "Database" {
                    tags "Observability"
                }
                prometheus = container "Prometheus" "Metrics store." "Prometheus" "Database" {
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
        cloudchamber.apiService -> cloudchamber.temporalServer "Observes and controls run executions via" "gRPC" "WorkflowClient"
        cloudchamber.apiService -> cloudchamber.artifactStore "Signs a short-lived read URL, and deletes an object the caller owns"
        endUser -> cloudchamber.artifactStore "Reads an owned file through a short-lived URL" "S3 API"
        cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording" "S3 API"
        cloudchamber.uploadService -> cloudchamber.apiService "Asks the API to check and record the file" "HTTPS/JSON"
        cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls workflow task queue and executes workflows via" "gRPC" "WorkflowPoll"
        cloudchamber.workflowWorker -> cloudchamber.postgres "Records steps of the sample workflow" "SQL/TCP" "PipelineData"
        cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls media task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.mediaWorker -> cloudchamber.postgres "Reads the source file, registers the audio, records steps" "SQL/TCP" "PipelineData"
        cloudchamber.mediaWorker -> cloudchamber.artifactStore "Checks the source object, reads it, and writes the audio object" "S3 API" "PipelineData"
        cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls frames task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.framesWorker -> cloudchamber.postgres "Registers the key frame, records its step" "SQL/TCP" "PipelineData"
        cloudchamber.framesWorker -> cloudchamber.artifactStore "Writes a key-frame object" "S3 API" "PipelineData"
        cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls ASR task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript, records its step" "SQL/TCP" "PipelineData"
        cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object" "S3 API" "PipelineData"
        cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls render task queue and executes activities via" "gRPC" "WorkflowPoll"
        cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document, records its step" "SQL/TCP" "PipelineData"
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
        endUser -> cloudchamber.apiService.runapp "Starts and controls runs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.auditapp "Uses audit APIs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.userapp "Manages users via" "HTTPS/JSON"
        cloudchamber.apiService.auditapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.userapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.projectapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.runapp -> cloudchamber.postgres "Stores runs and commands" "SQL/TCP"
        cloudchamber.apiService.runcontrol -> cloudchamber.postgres "Listens for run changes, reads specs, records status" "SQL/TCP"
        cloudchamber.apiService.runcontrol -> cloudchamber.temporalServer "Observes and controls run executions via" "gRPC"
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
                        deploymentNode "Pipeline workers" "One worker per task queue." "Workload group" "1" {
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
                        deploymentNode "workflow-worker" "Workflow code of every workflow." "Deployment" "1" {
                            containerInstance cloudchamber.workflowWorker
                        }
                        deploymentNode "media-worker" "Source and audio." "Deployment" "1" {
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
            description "The product path: API, auth, upload, storage, Temporal, and pipeline workers. Telemetry and platform databases are in the other views."
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
            description "LiteLLM and Langfuse only. ASR and render workers call LiteLLM from the product namespace (ContainersProduct); that edge is omitted here to avoid repeating the worker boxes."
            include operator
            include "element.tag==LLM"
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
            description "Grafana stack and cross-namespace scrape/export hooks from the API service. Worker OTLP fan-in is collapsed to the same single hub edge as ContainersProduct."
            include operator
            include "element.tag==Observability"
            include cloudchamber.apiService
            exclude "relationship.tag==TelemetryExport"
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
            description "A successful run of one recording. The API stores the run, the controller starts it, the workers run the steps and record them, and the controller records the result."

            endUser -> cloudchamber.apiService "Starts a run of the media2doc workflow for one file"
            cloudchamber.apiService -> cloudchamber.authAPI "Authenticates the caller" "HTTPS/JSON"
            cloudchamber.apiService -> cloudchamber.authAPI "Authorizes the caller" "HTTPS/JSON"
            cloudchamber.apiService -> cloudchamber.postgres "Stores the run in the pending phase, with the file id as its input"
            cloudchamber.apiService -> endUser "Returns the pending run"

            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller finds no execution for the run and starts the workflow"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the execution and the first workflow task"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as running"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the first workflow task and commands scheduling of ResolveSource"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ResolveSource task"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Checks the source file is ready and in the run's project"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Checks the source object"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Records the step as done"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ResolveSource result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ExtractAudio"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ExtractAudio task and heartbeats while it works"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Reads the source object and writes 16 kHz mono PCM audio"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Overwrites the step with how far it has got, at most once a second"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Registers the audio file in the run's project and records the step as done"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ExtractAudio result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task and heartbeats while it works"
            cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object"
            cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript in the run's project and records the step as done"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports the Transcribe result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "For a video, polls the next workflow task and commands scheduling of ExtractKeyFrames"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls the ExtractKeyFrames task and heartbeats while it works"
            cloudchamber.framesWorker -> cloudchamber.artifactStore "Writes a key frame"
            cloudchamber.framesWorker -> cloudchamber.postgres "Registers that frame in the run's project and records the step as done"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Reports the ExtractKeyFrames result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ComposeDocument"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls the ComposeDocument task and heartbeats while it works"
            cloudchamber.renderWorker -> cloudchamber.artifactStore "Writes the document object"
            cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document in the run's project and records the step as done"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Reports the ComposeDocument result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and completes the workflow with the file ids"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the completed workflow"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller sees the execution has completed and reads its result"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as succeeded with the file ids"

            endUser -> cloudchamber.apiService "Reads the run while it is moving, once a second"
            cloudchamber.apiService -> cloudchamber.postgres "Reads the run. It does not ask Temporal"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineFailureDynamic" {
            title "Dynamic View: A conversion fails, and is retried"
            description "A step fails for good, so the run fails. A retry continues from that step, and the steps before it do not run again."

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task"
            cloudchamber.asrWorker -> cloudchamber.postgres "Records the step as failed, with the message"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports the permanent Transcribe failure. That activity is not retried"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and returns that error"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the failed workflow. The registered audio stays"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller sees the execution has failed and reads the failure"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as failed with the step's message"

            endUser -> cloudchamber.apiService "Retries the run"
            cloudchamber.apiService -> cloudchamber.postgres "Raises the attempt the run is asked for by one, to continue from the failure"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller resets the execution to the workflow task that scheduled Transcribe"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Starts a new execution that keeps the history before that task"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the new attempt as running"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Replays the kept history and commands scheduling of Transcribe again"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task. ResolveSource and ExtractAudio do not run again"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineCancelDynamic" {
            title "Dynamic View: Suspend, resume, and cancel a conversion"
            description "Suspend stops the running step now, and resume continues it from where it stopped. Cancel stops the run for good."

            endUser -> cloudchamber.apiService "Suspends the run"
            cloudchamber.apiService -> cloudchamber.postgres "Records that the run is asked to suspend"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller pauses the execution, then the running Transcribe activity"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Heartbeats and learns of the pause"
            cloudchamber.asrWorker -> cloudchamber.postgres "Records the step as paused, with how far it had got"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as paused"

            endUser -> cloudchamber.apiService "Resumes the run"
            cloudchamber.apiService -> cloudchamber.postgres "Records that the run is no longer asked to suspend"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller unpauses the execution, then the activity"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls Transcribe again and continues from its last heartbeat"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as running"

            endUser -> cloudchamber.apiService "Cancels the run"
            cloudchamber.apiService -> cloudchamber.postgres "Records that the run is asked to cancel"
            cloudchamber.apiService -> cloudchamber.temporalServer "The run controller requests cancellation of the execution"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the cancellation request and a workflow task"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls that task and requests cancellation of Transcribe, then waits for it to close"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Heartbeats and learns the cancel"
            cloudchamber.asrWorker -> cloudchamber.postgres "Records the step as canceled"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Returns a canceled error after Transcribe closes. ExtractKeyFrames is not scheduled"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the canceled workflow"
            cloudchamber.apiService -> cloudchamber.postgres "The run controller records the run as canceled"

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
            description "Projects, accounts, runs and the controller that drives them, and the upload check inside the API."
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
            description "Local developer machine, browser tier, and Kind cluster. Each Kubernetes namespace is a child deployment node under the cluster."
            include *
            autoLayout tb
        }

        deployment * production "DeploymentProduction" {
            title "Deployment view: Production"
            description "Ingress, Kubernetes platform, and the same namespace boundaries as dev."
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
