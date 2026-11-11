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
                apiService = container "API service" "Primary HTTP API and domain routes. Starts pipeline workflows and reads their execution status and results." "Go" {
                    auditapp = component "auditapp" "Records account changes for an administrator to review." "Go"
                    userapp = component "userapp" "Creates and manages accounts." "Go"
                    projectapp = component "projectapp" "A person owns many projects. Each file and job belongs to one project." "Go"
                    pipelineapp = component "pipelineapp" "Starts a conversion from one file. Reads execution status and results from the Temporal frontend and copies them onto the job." "Go / Temporal client"
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
                temporalServer = container "Temporal Service" "Frontend, history, matching, and the internal worker service. Dev runs them in one auto-setup process. History and visibility are in Temporal PostgreSQL." "Temporal" "Service"
                temporalUI = container "Temporal UI" "Web interface for workflow visibility and operations." "Temporal UI" "Web Browser"
            }
            group "Pipeline execution" {
                workflowWorker = container "Workflow worker" "Runs ProcessSourceWorkflow on the workflow task queue. A pinned run stays on the build that started it. An auto-upgrade run moves to the target build." "Go / Temporal worker" "Service"
                mediaWorker = container "Media worker" "Source and audio activities on the media task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
                framesWorker = container "Frames worker" "Key-frame activities on the frames task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
                asrWorker = container "ASR worker" "Transcription activities on the ASR task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
                renderWorker = container "Render worker" "Compose and export activities on the render task queue. Dev runs a simulated stand-in." "Go / Temporal worker" "Service"
            }
            group "Artifact & data stores" {
                postgres = container "PostgreSQL" "Relational data for domains and migrations." "PostgreSQL" "Database"
                temporalPostgres = container "Temporal PostgreSQL" "Persistence and visibility storage for Temporal (physical isolation in dev)." "PostgreSQL" "Database"
                artifactStore = container "Artifact storage" "Object bytes for recordings and pipeline outputs. Project membership is the artifact row in PostgreSQL." "S3-compatible" "Database"
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
        cloudchamber.apiService -> cloudchamber.authAPI "Checks the caller through" "HTTPS/JSON"
        cloudchamber.authAPI -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.metricsAPI -> cloudchamber.apiService "Scrapes metrics from" "HTTP"
        cloudchamber.metricsAPI -> cloudchamber.workflowWorker "Scrapes poll and schedule-to-start metrics from" "HTTP"
        cloudchamber.metricsAPI -> cloudchamber.mediaWorker "Scrapes poll and schedule-to-start metrics from" "HTTP"
        cloudchamber.metricsAPI -> cloudchamber.framesWorker "Scrapes poll and schedule-to-start metrics from" "HTTP"
        cloudchamber.metricsAPI -> cloudchamber.asrWorker "Scrapes poll and schedule-to-start metrics from" "HTTP"
        cloudchamber.metricsAPI -> cloudchamber.renderWorker "Scrapes poll and schedule-to-start metrics from" "HTTP"
        operator -> cloudchamber.temporalUI "Inspects workflows in" "HTTPS"
        cloudchamber.temporalUI -> cloudchamber.temporalServer "Connects to" "gRPC"
        cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService -> cloudchamber.temporalServer "Starts workflows and reads execution status and results via" "gRPC"
        cloudchamber.apiService -> cloudchamber.artifactStore "Signs a short-lived read URL, and deletes an object the caller owns"
        endUser -> cloudchamber.artifactStore "Reads an owned file through a short-lived URL" "S3 API"
        cloudchamber.uploadService -> cloudchamber.artifactStore "Stores the recording" "S3 API"
        cloudchamber.uploadService -> cloudchamber.apiService "Asks the API to check and record the file" "HTTPS/JSON"
        cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls workflow task queue and executes workflows via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls media task queue and executes activities via" "gRPC"
        cloudchamber.mediaWorker -> cloudchamber.postgres "Loads the source file and registers the audio file in its project" "SQL/TCP"
        cloudchamber.mediaWorker -> cloudchamber.artifactStore "Checks the source object and writes the audio object" "S3 API"
        cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls frames task queue and executes activities via" "gRPC"
        cloudchamber.framesWorker -> cloudchamber.postgres "Registers a key frame in the source file's project" "SQL/TCP"
        cloudchamber.framesWorker -> cloudchamber.artifactStore "Writes a key-frame object" "S3 API"
        cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls ASR task queue and executes activities via" "gRPC"
        cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript in the source file's project" "SQL/TCP"
        cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object" "S3 API"
        cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls render task queue and executes activities via" "gRPC"
        cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document in the source file's project" "SQL/TCP"
        cloudchamber.renderWorker -> cloudchamber.artifactStore "Writes the document object" "S3 API"

        // C3 — API service (subdomains aligned to current packages)
        endUser -> cloudchamber.apiService.userapp "Uses user APIs on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.projectapp "Creates and opens projects on" "HTTPS/JSON"
        endUser -> cloudchamber.apiService.pipelineapp "Starts a conversion from a file id on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.auditapp "Uses audit APIs on" "HTTPS/JSON"
        cloudchamber.adminUI -> cloudchamber.apiService.userapp "Manages users via" "HTTPS/JSON"
        cloudchamber.apiService.auditapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.userapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.projectapp -> cloudchamber.postgres "Reads from and writes to" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.postgres "Stores the job in the source file's project and copies execution status onto it" "SQL/TCP"
        cloudchamber.apiService.pipelineapp -> cloudchamber.temporalServer "Starts workflows and reads execution status and results via" "gRPC"
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
                        deploymentNode "Pipeline workers" "One worker for each step. A crash waits until that worker polls again." "Workload group" "1" {
                            deploymentNode "Workload: workflow-worker" "Workflow." "Deployment" "1" {
                                containerInstance cloudchamber.workflowWorker
                            }
                            deploymentNode "Workload: media-worker" "Audio." "Deployment" "1" {
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

                deploymentNode "Pipeline workers" "Two workers of the same build on each task queue." "Workload group" "1" {
                    deploymentNode "workflow-workers" "Two workers of one build. A pinned run stays on its starting build." "Deployment" 2 {
                        containerInstance cloudchamber.workflowWorker
                    }
                    deploymentNode "media-workers" "Two of the same build." "Deployment" 2 {
                        containerInstance cloudchamber.mediaWorker
                    }
                    deploymentNode "frames-workers" "Two of the same build." "Deployment" 2 {
                        containerInstance cloudchamber.framesWorker
                    }
                    deploymentNode "asr-workers" "Two of the same build." "Deployment" 2 {
                        containerInstance cloudchamber.asrWorker
                    }
                    deploymentNode "render-workers" "Two of the same build." "Deployment" 2 {
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

        dynamic cloudchamber "PipelineProcessDynamic" {
            title "Dynamic View: Convert a recording"
            description "Successful run of one file, in this order. The API checks the caller, loads the source file, records the job, and starts ProcessSourceWorkflow. History records the execution and schedules the first workflow task, and that start call then returns. The API stores the run id and returns the job. The workflow worker polls the first workflow task after it is queued. That poll may overlap the API response. That same task records cancel-waits version 2, then schedules ResolveSource. Each later activity waits for the previous result and is scheduled by its own workflow task: ExtractAudio on media, Transcribe on ASR, ExtractKeyFrames on frames, ComposeExport on render. The task that schedules ExtractKeyFrames records the frames-queue version first. ResolveSource loads the source row and checks the object. Each writing step stores its object, then registers it in the source file's project. A video records a key frame after the transcript. Audio reports that step with no frame. The next workflow task commands completion. Listing and opening the job are later requests, not spans in the workflow trace. A list reads the job rows and does not ask the frontend. Opening a job whose row is still open reads the execution. A completed execution also stores the output file ids. A failed execution stores the failed status. A row that is already closed is not read from the frontend again."

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
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Writes the audio object"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Registers the audio file in the source file's project"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ExtractAudio result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task"
            cloudchamber.asrWorker -> cloudchamber.artifactStore "Writes the transcript object"
            cloudchamber.asrWorker -> cloudchamber.postgres "Registers the transcript in the source file's project"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports the Transcribe result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task, records the frames-queue version, and commands scheduling of ExtractKeyFrames"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Polls the ExtractKeyFrames task"
            cloudchamber.framesWorker -> cloudchamber.artifactStore "For a video, writes a key frame"
            cloudchamber.framesWorker -> cloudchamber.postgres "For a video, registers that frame in the source file's project"
            cloudchamber.framesWorker -> cloudchamber.temporalServer "Reports the ExtractKeyFrames result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of ComposeExport"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Polls the ComposeExport task"
            cloudchamber.renderWorker -> cloudchamber.artifactStore "Writes the document object"
            cloudchamber.renderWorker -> cloudchamber.postgres "Registers the document in the source file's project"
            cloudchamber.renderWorker -> cloudchamber.temporalServer "Reports the ComposeExport result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands completion of the workflow"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the completed workflow"

            endUser -> cloudchamber.apiService "Lists jobs in the project"
            cloudchamber.apiService -> cloudchamber.postgres "Reads the job rows"

            endUser -> cloudchamber.apiService "Opens the job"
            cloudchamber.apiService -> cloudchamber.temporalServer "Reads the execution while the job row is still open"
            cloudchamber.apiService -> cloudchamber.postgres "Stores that status. A completed execution also stores the output file ids"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineFailureDynamic" {
            title "Dynamic View: A conversion fails"
            description "A transient activity error is retried on the same activity. A permanent activity error is not retried. This run hits that permanent failure after audio was stored. The workflow returns that error and closes as failed. The audio stays registered. Opening the job while the row is still open reads the execution and the progress query, then stores failed, the activity error, and the file ids already held."

            endUser -> cloudchamber.apiService "Starts a conversion from one file"
            cloudchamber.apiService -> cloudchamber.postgres "Loads the source file"
            cloudchamber.apiService -> cloudchamber.postgres "Records the job with that file's owner and project"
            cloudchamber.apiService -> cloudchamber.temporalServer "Starts ProcessSourceWorkflow with the job and file ids"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the execution and the first workflow task"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the workflow task and commands scheduling of ExtractAudio"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Polls the ExtractAudio task"
            cloudchamber.mediaWorker -> cloudchamber.artifactStore "Writes the audio object"
            cloudchamber.mediaWorker -> cloudchamber.postgres "Registers the audio file in the source file's project"
            cloudchamber.mediaWorker -> cloudchamber.temporalServer "Reports the ExtractAudio result"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and commands scheduling of Transcribe"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Polls the Transcribe task"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Reports a permanent Transcribe failure. That activity is not retried"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls the next workflow task and returns that error"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the failed workflow. The registered audio stays"

            endUser -> cloudchamber.apiService "Opens the job"
            cloudchamber.apiService -> cloudchamber.temporalServer "Reads the failed execution and its progress while the job row is still open"
            cloudchamber.apiService -> cloudchamber.postgres "Stores failed, the activity error, and the file ids already held"

            autoLayout tb
        }

        dynamic cloudchamber "PipelineCancelDynamic" {
            title "Dynamic View: Cancel a conversion"
            description "Cancel arrives while Transcribe is in flight, after audio was stored. The API requests cancellation and leaves the job row open. A new run records cancel-waits version 2 and waits for Transcribe to close. If Transcribe stops on the cancel, it does not write. If Transcribe completes anyway, its file id stays on the progress query and the workflow returns the cancel error before scheduling ExtractKeyFrames. The execution closes as canceled. Returning normally would close it as completed. Returning any other error would close it as failed. Version 1 waits and can still schedule the next activity. A run started before version 1 returns when the request arrives. Opening the job while the row is still open reads the execution and the progress query, then stores canceled, the activity name, and the file ids already held. A later open reads that row and does not ask the frontend."

            endUser -> cloudchamber.apiService "Cancels the job"
            cloudchamber.apiService -> cloudchamber.temporalServer "Requests cancellation of the execution"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the cancellation request and a workflow task"

            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Polls that task and requests cancellation of Transcribe, then waits for it to close"
            cloudchamber.asrWorker -> cloudchamber.temporalServer "Heartbeats, learns the cancel, and stops before writing the transcript"
            cloudchamber.workflowWorker -> cloudchamber.temporalServer "Returns a canceled error after Transcribe closes"
            cloudchamber.temporalServer -> cloudchamber.temporalPostgres "Records the canceled workflow"

            endUser -> cloudchamber.apiService "Opens the job"
            cloudchamber.apiService -> cloudchamber.temporalServer "Reads the canceled execution and its progress while the job row is still open"
            cloudchamber.apiService -> cloudchamber.postgres "Stores canceled, the activity name, and the audio file id"

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
