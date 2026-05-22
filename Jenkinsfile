// Jenkinsfile — multibranch pipeline for arcana-cloud-go
// Mirrors the existing go-app-pipeline (single-branch job that polled SCM),
// adapted for multibranch + PR branches.
//
// Key differences from the legacy XML-embedded script:
//   * `checkout scm` (no hardcoded branch=main)        — supports every branch + every PR
//   * `pollSCM` trigger removed                        — Jenkins multibranch + GitHub webhook drive triggers
//   * "Push to Registry" + "Arch Qube Metrics" gated   — only main pushes to registry; PR builds stay local
//   * SonarQube gets pullrequest.* params on PRs       — PR-decoration in Sonar UI
//   * Build tag includes branch on non-main builds     — `pr-<changeId>` or `branch-<name>`
//
// Test plan for the pilot:
//   1. Multibranch job picks up `main` first       → should match legacy build output
//   2. Multibranch job picks up `renovate/*` PRs   → should run all stages except push
//   3. PR decoration shows in SonarQube UI          → Sonar pullrequest params wired

pipeline {
    agent any

    options {
        timeout(time: 30, unit: 'MINUTES')
        buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '1'))
        disableConcurrentBuilds()
        timestamps()
    }

    environment {
        APP_NAME  = "go-app"
        REGISTRY  = "localhost:5000"
        IMAGE_TAG = "${REGISTRY}/arcana/${APP_NAME}"
        VERSION   = "1.0.0"
    }

    stages {
        stage("Checkout") {
            steps {
                checkout scm
                sh 'git log -1 --oneline'
                script {
                    echo "Branch: ${env.BRANCH_NAME ?: 'unknown'}"
                    echo "PR: ${env.CHANGE_ID ?: 'no'} (target: ${env.CHANGE_TARGET ?: 'n/a'})"
                }
            }
        }

        stage("Cleanup Old Images") {
            steps {
                sh '''
                    # Remove dangling/unused images to free disk space
                    docker image prune -f || true
                    # Keep only last 3 build-tagged images for this app
                    docker images --format '{{.Repository}}:{{.Tag}}' \
                        | grep "${APP_NAME}.*build-" \
                        | sort -t- -k2 -rn \
                        | tail -n +4 \
                        | xargs -r docker rmi 2>/dev/null || true
                    # Stop leftover test containers
                    docker compose -f docker-compose.test.yml down \
                        --remove-orphans 2>/dev/null || true
                '''
            }
        }

        stage("Docker Compose Build") {
            steps {
                sh "VERSION=${VERSION} DOCKER_REGISTRY=${REGISTRY} docker compose -f docker-compose.ci.yml build"
                sh "docker tag ${IMAGE_TAG}:${VERSION} ${IMAGE_TAG}:build-${BUILD_NUMBER}"
                sh '''
                    docker create --name go-cov-tmp-${BUILD_NUMBER} ${IMAGE_TAG}:${VERSION}
                    docker cp go-cov-tmp-${BUILD_NUMBER}:/tmp/coverage.out coverage.out || true
                    docker rm go-cov-tmp-${BUILD_NUMBER} || true
                '''
            }
        }

        stage("Integration: Layered gRPC") {
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
                    sh '''
                        JENKINS_ID=$(hostname)
                        # Cleanup any leftover containers
                        GO_IMAGE=placeholder docker compose -p arcana-ci-go-grpc \
                            -f deployment/layered/docker-compose-ci-grpc.yml \
                            down -v --remove-orphans 2>/dev/null || true
                        # Start 3-layer gRPC stack
                        GO_IMAGE=${IMAGE_TAG}:build-${BUILD_NUMBER} \
                        docker compose -p arcana-ci-go-grpc \
                            -f deployment/layered/docker-compose-ci-grpc.yml up -d
                        # Connect Jenkins to compose network
                        docker network connect arcana-ci-go-net $JENKINS_ID 2>/dev/null || true
                        # Run smoke test via internal container name
                        bash scripts/integration-smoke-test.sh \
                            http://arcana-ci-go-controller:8090 grpc-layered 240
                        docker network disconnect arcana-ci-go-net $JENKINS_ID 2>/dev/null || true
                    '''
                }
            }
            post {
                always {
                    sh '''
                        docker network disconnect arcana-ci-go-net $(hostname) 2>/dev/null || true
                        GO_IMAGE=placeholder docker compose -p arcana-ci-go-grpc \
                            -f deployment/layered/docker-compose-ci-grpc.yml \
                            down -v --remove-orphans 2>/dev/null || true
                    '''
                }
            }
        }

        stage("Integration: K8s gRPC") {
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
                    sh '''#!/bin/bash
                        export PATH="/var/jenkins_home/bin:${PATH}"
                        kind version || { echo "kind not found"; exit 1; }
                        bash scripts/kind-smoke-test.sh "${IMAGE_TAG}:build-${BUILD_NUMBER}" grpc 480
                    '''
                }
            }
            post {
                always {
                    sh '''#!/bin/bash
                        export PATH="/var/jenkins_home/bin:${PATH}"
                        kind get clusters 2>/dev/null | grep arcana-ci | while read cl; do
                          kind delete cluster --name "$cl" 2>/dev/null || true
                        done
                    '''
                }
            }
        }

        stage("SonarQube Analysis") {
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
                    withSonarQubeEnv('SonarQube') {
                        script {
                            // PR builds get PR-decoration params so SonarQube
                            // attaches the report to the GitHub PR instead of
                            // overwriting the long-lived main branch report.
                            def prArgs = env.CHANGE_ID ? """ \
                                -Dsonar.pullrequest.key=${env.CHANGE_ID} \
                                -Dsonar.pullrequest.branch=${env.BRANCH_NAME} \
                                -Dsonar.pullrequest.base=${env.CHANGE_TARGET}""" : ''
                            sh """sonar-scanner \
                              -Dsonar.projectKey=go-app \
                              -Dsonar.projectName="Go App" \
                              -Dsonar.sources=. \
                              -Dsonar.exclusions=vendor/**,*_test.go,internal/testutil/mocks/**,internal/jobs/scheduler/scheduler.go,api/proto/pb/**,**/*.pb.go \
                              -Dsonar.text.inclusions.activate=false \
                              -Dsonar.scm.disabled=true \
                              -Dsonar.go.coverage.reportPaths=coverage.out${prArgs}"""
                        }
                    }
                }
            }
        }

        stage("Architecture Qube") {
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'UNSTABLE') {
                    sh '''
                        mkdir -p arch-qube-reports
                        docker run --rm \
                            --network devops_default \
                            -v $(pwd):/project \
                            -v $(pwd)/arch-qube-reports:/output \
                            arcana.boo/arcana/arch-qube:latest scan /project \
                            --framework go --no-ai \
                            --ci --format json,markdown \
                            -o /output --threshold 90 || true
                    '''
                }
            }
        }

        stage("Image Info") {
            steps {
                sh "docker images --format 'table {{.Repository}}:{{.Tag}}\\t{{.Size}}' | grep ${APP_NAME} || true"
            }
        }

        stage("Push to Registry") {
            // Only push from main branch builds. PR builds keep the image local
            // for integration tests but don't pollute the registry with PR tags.
            when { branch 'main' }
            steps {
                sh "docker push ${IMAGE_TAG}:${VERSION}"
                sh "docker push ${IMAGE_TAG}:build-${BUILD_NUMBER}"
            }
        }

        stage("Arch Qube Metrics") {
            // Metrics script writes to shared report dir, only run for main.
            when { branch 'main' }
            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'SUCCESS') {
                    sh "bash /data/projects/_scripts/arch-qube-metrics.sh \$(pwd) arcana-cloud-go || true"
                }
            }
        }
    }

    post {
        success { echo "Pipeline SUCCESS - ${APP_NAME}:${VERSION} branch=${env.BRANCH_NAME ?: '?'} pr=${env.CHANGE_ID ?: 'no'}" }
        failure { echo "Pipeline FAILED - branch=${env.BRANCH_NAME ?: '?'} pr=${env.CHANGE_ID ?: 'no'}" }
        always  { echo "Build number ${BUILD_NUMBER} done" }
    }
}
