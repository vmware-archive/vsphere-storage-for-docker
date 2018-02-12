pipeline {
    agent none

    stages {
        stage('Checkout code') {
            failFast true
            parallel {
                stage('Checkout for ESX 6.5 runs') {
                /* Let's make sure we have the repository cloned to our workspace. */
                    agent {
                        label "vdvs-65-slaves"
                    }
                    steps {
                        checkout scm
                    }
                }
                stage('Checkout for ESX 6.0 runs') {
                /* Let's make sure we have the repository cloned to our workspace..*/
                    agent {
                        label "vdvs-60-slaves"
                    }
                    steps {
                        checkout scm
                    }
                }
            }
        }

        stage('Build binaries') {
        /* This builds VDVS binaries */
            failFast true
            parallel {
                stage('Build binaries for ESX 6.5 runs') {
                    agent {
                        label "vdvs-65-slaves"
                    }
                    steps {
                        sh "echo Building the VDVS binaries"
                        sh "export PKG_VERSION=$BUILD_NUMBER"
                        sh "make build-all"
                    }
                }

                stage('Build binaries for ESX 6.0 runs') {
                    agent {
                        label "vdvs-60-slaves"
                    }
                    steps {
                        sh "echo Building the VDVS binaries"
                        sh "export PKG_VERSION=$BUILD_NUMBER"
                        sh "make build-all"
                    }
                }   
            }
        }

        stage('Deployment') {
            failFast true
            parallel {
                stage('Deploy binaries for ESX 6.5 runs') {
                    agent {
                        label "vdvs-65-slaves"
                    }
                    steps {
                        sh "echo Deployment On 6.5 setup"
                        sh "echo ESX=$ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3; echo PKG_VERSION"
                        sh "echo deploying binaries"
                        sh "make deploy-all"
                        sh "echo finished deploying the binaries"
                    }
                }

                stage('Deploy binaries for ESX 6.0 runs') {
                    agent {
                        label "vdvs-60-slaves"
                    }
                    steps {
                        sh "echo Deployment On 6.0 setup"
                        sh "echo ESX=$ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3; echo PKG_VERSION"
                        sh "echo deploying binaries"
                        sh "make deploy-all" 
                        sh "echo finished deploying the binaries"
                    }
                }
            }
        }

        stage('Test VDVS') {
            failFast true
            parallel {
                stage('Run tests on ESX 6.5') {
                    agent {
                        label "vdvs-65-slaves"
                    }

                steps {
                    sh "echo Test VDVS On 6.5 setup"
                    sh "echo ESX=$ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3; echo PKG_VERSION"
                    sh "echo starting e2e tests" 
                    sh "make test-e2e"
                    sh "make test-esx"
                    sh "make test-vm"
                    }
                }
                stage('Run tests on ESX 6.0') {
                    agent {
                    label "vdvs-60-slaves"
                    }   
                steps {
                    sh "echo Test VDVS On 6.0 setup"
                    sh "echo ESX=$ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3; echo PKG_VERSION"
                    sh "echo starting e2e tests"
                    sh "make test-e2e"
                    sh "make test-esx"
                    sh "make test-vm"
                    }
                }
            }
        }
   
        stage('Test vFile') {
            failFast true
            parallel {
                stage('Run vFile tests on ESX 6.5') {
                    agent {
                       label "vdvs-65-slaves"
                    }
                    steps {
                        script{
                            def stopContainers = "docker stop \$(docker ps -a -q) 2> /dev/null || true"
                            def removeContainers = "docker rm \$(docker ps -a -q) 2> /dev/null || true"
                            def removeVolumes =  "docker volume rm \$(docker volume ls -q -f dangling=true) 2> /dev/null || true"
                            try{
                                sh "echo Build, deploy, and test vFile on 6.5 setup"
                                sh "echo Build vFile binaries"
                                sh "echo ESX = $ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3;" 
                                sh "make build-vfile-all" 
                                sh "echo Deploy the vFile binaries"
                                sh "make deploy-vfile-plugin"
                                sh "echo Start the vFile tests"
                                sh "make test-e2e-vfile"
                                sh "echo vFile tests finished"  
                            } finally{
                                sh "ssh ${env.GOVC_USERNAME}@$VM1 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "ssh ${env.GOVC_USERNAME}@$VM2 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "ssh ${env.GOVC_USERNAME}@$VM3 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "make clean-vfile"
                                sh "make clean-all"
                             } 
                        }
                    }
                }
      
                stage('Run vFile tests on ESX 6.0') {
                    agent {
                        label "vdvs-60-slaves"
                    }

                    steps {
                        script{
                            def stopContainers = "docker stop \$(docker ps -a -q) 2> /dev/null || true"
                            def removeContainers = "docker rm \$(docker ps -a -q) 2> /dev/null || true"
                            def removeVolumes =  "docker volume rm \$(docker volume ls -q -f dangling=true) 2> /dev/null || true"
                            try{
                                echo "Build, deploy, and test vFile on 6.0 setup"
                                sh "echo Build vFile binaries"
                                sh "echo ESX=$ESX; echo VM1=$VM1; echo VM2=$VM2; echo VM3=$VM3;" 
                                sh "make build-vfile-all" 
                                sh "echo Deploy the vFile binaries"
                                sh "make deploy-vfile-plugin"
                                sh "echo Run the vFile tests"
                                sh "make test-e2e-vfile"
                                sh "echo vFile tests finished"
                            }finally{
                                sh "ssh ${env.GOVC_USERNAME}@$VM1 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "ssh ${env.GOVC_USERNAME}@$VM2 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "ssh ${env.GOVC_USERNAME}@$VM3 ${stopContainer}; ${removeContainer}; ${removeVolume}"
                                sh "make clean-vfile"
                                sh "make clean-all"   
                            } 
                        }

                    }
                }
            }
        }

        stage('Test Windows plugin') {
        /* This builds, deploys and tests the windows binaries */
            agent {
                        label "vdvs-65-slaves"
                    }

            steps {
                sh "echo Build, deploy, and test Windows plugin"
                sh "echo Windows-VM=$WIN_VM1"
                sh "echo Build Windows plugin binaries"
                sh "make build-windows-plugin"
                sh "echo Finished building binaries for windows"
                sh "echo Deploy the Windows plugin binaries"
                sh "make deploy-windows-plugin"
                sh "echo Finished deploying binaries for windows"
                sh "echo Run the Windows plugin tests"
                sh "make test-e2e-windows"
                sh "echo Windows plugin tests finished"
            }
        }
    }

    // The options directive is for configuration that applies to the whole job.
    options {
        // This ensures we only have 10 builds at a time, so
        // we don't fill up our storage!
        buildDiscarder(logRotator(numToKeepStr:'10'))
    }
}