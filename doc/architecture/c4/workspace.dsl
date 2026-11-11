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
                    auditapp = component "auditapp" "Records account changes for an administrator to review." "Go"
                    userapp = component "userapp" "Creates and manages accounts." "Go"
                    projectapp = component "projectapp" "A person owns many projects. Each file and job belongs to one project." "Go"
                    pipelineapp = component "pipelineapp" "Starts a conversion from one file and keeps that file's project." "Go / Temporal client"
                    uploadapp = component "uploadapp" "Checks the project on an upload, records the file when the upload finishes, and refuses to remove a finished file." "Go"
                }
                authAPI = container "Auth service" "JWT issue and validation." "Go" {
                    authapp = component "authapp" "Issues a sign-in token and checks it." "Go"
                    grpcauthapp = component "grpcauthapp" "Checks a token for another service." "Go"
                    userbus = component "userbus" "The account record used to sign in." "Go"
                }
                uploadService = container "Upload service" "Accepts a recording and stores it." "tusd" "Service"
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
                artifactStore = container "Artifact storage" "Holds each project's recordings and documents." "S3-compatible" "Database"
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
        endUser -> cloudchamber.uploadService "Uploads a recording" "HTTPS"
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
        cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording" "S3 API"
        cloudchamber.uploadService -> cloudchamber.apiService "Asks the API to check and record the file" "HTTPS/JSON"
        cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls workflow task queue and executes workflows via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls media task queue and executes activities via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.artifactStore "Reads the source and writes audio and key frames in the same project"
        cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls ASR task queue and executes activities via" "gRPC"
        cloudchamber.asrWorker -> cloudchamber.artifactStore "Reads audio and writes the transcript in the same project"
        cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls render task queue and executes activities via" "gRPC"
        cloudchamber.renderWorker -> cloudchamber.artifactStore "Reads the inputs and writes the document in the same project"

        // C3 — API service (subdomains aligned to current packages)
        endUser -> cloudchamber.apiService.userapp "Uses user APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.projectapp "Creates and opens projects on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.pipelineapp "Starts a conversion from a file id on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.auditapp "Uses audit APIs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.userapp "Manages users via" "HTTPS/JSON"
        cloudchamber.apiService.auditapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.userapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.projectapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.postgres "Copies the source file's owner and project onto the job" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.temporalServer "Starts and queries workflows via" "gRPC"
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
                        deploymentNode "Pipeline orchestration" "Workflow control." "Workload group" "1" {
                            deploymentNode "Workload: temporal-server" "Workflow server." "Deployment" "1" {
                                containerInstance cloudchamber.temporalServer
                            }
                            deploymentNode "Workload: temporal-ui" "Workflow view." "Deployment" "1" {
                                containerInstance cloudchamber.temporalUI
                            }
                        }
                        deploymentNode "Artifact & data stores" "Databases and files." "Workload group" "1" {
                            deploymentNode "Workload: database" "Application data." "StatefulSet" "1" {
                                containerInstance cloudchamber.postgres
                            }
                            deploymentNode "Workload: temporal-database" "Workflow data." "StatefulSet" "1" {
                                containerInstance cloudchamber.temporalPostgres
                            }
                            deploymentNode "Workload: seaweedfs" "Files." "Deployment" "1" {
                                containerInstance cloudchamber.artifactStore
                            }
                        }
                        deploymentNode "Pipeline workers" "One worker for each step." "Workload group" "1" {
                            deploymentNode "Workload: workflow-worker" "Workflow." "Deployment" "1" {
                                containerInstance cloudchamber.workflowWorker
                            }
                            deploymentNode "Workload: media-worker" "Audio and frames." "Deployment" "1" {
                                containerInstance cloudchamber.mediaWorker
                            }
                            deploymentNode "Workload: asr-worker" "Transcript." "Deployment" "1" {
                                containerInstance cloudchamber.asrWorker
                            }
                            deploymentNode "Workload: render-worker" "Document." "Deployment" "1" {
                                containerInstance cloudchamber.renderWorker
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
                deploymentNode "Application tier" "API, auth, and upload." "Workload group" "1" {
                    deploymentNode "admin-ui" "Admin site." "Deployment" "1" {
                        containerInstance cloudchamber.adminUI
                    }
                    deploymentNode "api-service" "API." "Deployment" 1 {
                        containerInstance cloudchamber.apiService
                    }
                    deploymentNode "auth-service" "Sign-in." "Deployment" 1 {
                        containerInstance cloudchamber.authAPI
                    }
                    deploymentNode "upload-service" "Uploads." "Deployment" 1 {
                        containerInstance cloudchamber.uploadService
                    }
                    deploymentNode "metrics" "Metrics." "Deployment" "1" {
                        containerInstance cloudchamber.metricsAPI
                    }
                }

                deploymentNode "Pipeline orchestration" "Workflow control." "Workload group" "1" {
                    deploymentNode "temporal-service" "Workflow server." "Temporal" 1 {
                        containerInstance cloudchamber.temporalServer
                    }
                    deploymentNode "temporal-ui" "Workflow view." "Deployment" "1" {
                        containerInstance cloudchamber.temporalUI
                    }
                }

                deploymentNode "Pipeline workers" "One pool for each step." "Workload group" "1" {
                    deploymentNode "workflow-workers" "Workflow." "Deployment" 1 {
                        containerInstance cloudchamber.workflowWorker
                    }
                    deploymentNode "media-workers" "Audio and frames." "Deployment" 1 {
                        containerInstance cloudchamber.mediaWorker
                    }
                    deploymentNode "asr-workers" "Transcript." "Deployment" 1 {
                        containerInstance cloudchamber.asrWorker
                    }
                    deploymentNode "render-workers" "Document." "Deployment" 1 {
                        containerInstance cloudchamber.renderWorker
                    }
                }

                deploymentNode "Artifact & data stores" "Databases and files." "Workload group" "1" {
                    deploymentNode "application-db" "Application data." "PostgreSQL" "1" {
                        containerInstance cloudchamber.postgres
                    }
                    deploymentNode "temporal-persistence" "Workflow data." "PostgreSQL" "1" {
                        containerInstance cloudchamber.temporalPostgres
                    }
                    deploymentNode "artifact-storage" "Files." "S3-compatible" "1" {
                        containerInstance cloudchamber.artifactStore
                    }
                }

                deploymentNode "Observability" "Logs, metrics, and traces." "Platform services" "1" {
                    infrastructureNode "Telemetry" "Logs, metrics, and traces." "Observability"
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
            description "Projects, accounts, conversions, and the upload check inside the API."
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
            description "The local cluster."
            include *
            autoLayout tb
        }

        deployment * production "DeploymentProduction" {
            title "Deployment view: Production"
            description "The same services, run for production."
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
