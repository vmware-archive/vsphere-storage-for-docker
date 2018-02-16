#!/usr/bin/groovy

VDVS_65_NODE_NAME = ""
VDVS_65_NODE_ID = ""
VDVS_60_NODE_NAME = ""
VDVS_60_NODE_ID = ""

pipeline {
    agent none

    stages {
        stage('Slave selection') {
        failFast true
        parallel {
        stage('Select slave for ESX 6.5 runs') {
            agent {
                        label "vdvs-65-slaves && available"
            }
            steps {
            script {
                VDVS_65_NODE_NAME = env.NODE_NAME
                VDVS_65_NODE_ID = UUID.randomUUID().toString()
                replaceNodeLabel(VDVS_65_NODE_NAME, "available", VDVS_65_NODE_ID)
            }
            sleep 2
            }
        }
        stage('Select slave for ESX 6.0 runs') {
            agent {
                        label "vdvs-60-slaves && available"
            }
            steps {
            script {
                            VDVS_60_NODE_NAME = env.NODE_NAME
                            VDVS_60_NODE_ID = UUID.randomUUID().toString()
                            replaceNodeLabel(VDVS_60_NODE_NAME, "available", VDVS_60_NODE_ID)
            }
            sleep 2
            }
        }
        }
    }

        stage('Checkout code') {
            failFast true
            parallel {
                stage('Checkout for ESX 6.5 runs') {
                /* Let's make sure we have the repository cloned to our workspace. */
                    steps {
                        node (VDVS_65_NODE_ID as String) { 
                            echo "Node name: " + env.NODE_NAME + " Node Id: " +VDVS_65_NODE_ID
                            checkout scm
                        }
                    }
                }
                stage('Checkout for ESX 6.0 runs') {
                /* Let's make sure we have the repository cloned to our workspace..*/
                    
                    steps {
                        node (VDVS_60_NODE_ID as String) { 
                            echo "Node name: " + env.NODE_NAME + " Node Id: " +VDVS_60_NODE_ID
                            checkout scm
                        }
                    }
                }
            }
        }

        stage('Build binaries') {
        /* This builds VDVS binaries */
            failFast true
            parallel {
                stage('Build binaries for ESX 6.5 runs') {
                    
                    steps {
                        node (VDVS_65_NODE_ID as String) { 
                            echo "Building the VDVS binaries"
                            sh "export PKG_VERSION=$BUILD_NUMBER"
                            sh "make build-all"
                        }
                    }
                }

                stage('Build binaries for ESX 6.0 runs') {
                    steps {
                        node (VDVS_60_NODE_ID as String) { 
                            echo "Building the VDVS binaries"
                            sh "export PKG_VERSION=$BUILD_NUMBER"
                            sh "make build-all"
                        }
                    }
                }   
            }
        }

        stage('Deployment') {
            failFast true
            parallel {
                stage('Deploy binaries for ESX 6.5 runs') {
                    steps {
                        node (VDVS_65_NODE_ID as String) { 
                            echo "Deployment On 6.5 setup"
                            echo "ESX=$ESX; VM1=$VM1; VM2=$VM2; VM3=$VM3; PKG_VERSION"
                            sh "make deploy-all"
                            echo "Finished deploying the binaries"
                        }
                    }
                }

                stage('Deploy binaries for ESX 6.0 runs') {
                    steps {
                        node (VDVS_60_NODE_ID as String) { 
                            echo "Deployment On 6.0 setup"
                            echo "ESX=$ESX; VM1=$VM1; VM2=$VM2; VM3=$VM3; PKG_VERSION"
                            echo "Deploying binaries"
                            sh "make deploy-all" 
                            echo "Finished deploying the binaries"
                        }
                    }
                }
            }
        }

        stage('Test VDVS') {
            failFast true
            parallel {
                stage('Run tests on ESX 6.5') {
                    steps {
                        node (VDVS_65_NODE_ID as String) { 
                             script {
                                try {
                                    echo "Test VDVS On 6.5 setup"
                                    echo " ESX=$ESX, VM1=$VM1, VM2=$VM2, VM3=$VM3, PKG_VERSION"
                                    echo " starting e2e tests"
                                    sh "make test-e2e"
                                    sh "make test-esx"
                                    sh "make test-vm"
                                }
                                catch (ex) {
                                    def cleanVfile= "False"
                                    cleanSetup(cleanVfile)
                                    throw ex
                                }
                            }
                        }
                   }
               }
        
               stage('Run tests on ESX 6.0') {
                   steps {
                       node (VDVS_60_NODE_ID as String) { 
                               script {
                                try {
                                    echo "Test VDVS On 6.0 setup"
                                    echo "ESX=$ESX, VM1=$VM1, VM2=$VM2, echo VM3=$VM3, echo PKG_VERSION"
                                    echo "Starting e2e tests"
                                    sh "make test-e2e"
                                    sh "make test-esx"
                                    sh "make test-vm"
                                }
                                catch (ex) {
                                    def cleanVfile = "False"
                                    cleanSetup(cleanVfile)
                                    throw ex
                                }
                            }
                       }
                    }
                }
            }
        }

        stage('Test vFile') {
            failFast true
            parallel {
                stage('Run vFile tests on ESX 6.5') {                  
                    steps {
                        node (VDVS_65_NODE_ID as String) { 
                            script {
                                try {
                                    echo "Build, deploy, and test vFile on 6.5 setup"
                                    echo "Build vFile binaries"
                                    echo "ESX = $ESX, VM1=$VM1, VM2=$VM2, VM3=$VM3;" 
                                    sh "make build-vfile-all" 
                                    echo " Deploy the vFile binaries"
                                    sh "make deploy-vfile-plugin"
                                    echo "Start the vFile tests"
                                    sh "make test-e2e-vfile"
                                    echo "vFile tests finished"  
                                } finally {
                                    def cleanVfile = "True" 
                                    cleanSetup(cleanVfile)
                                } 
                            }
                        }
                        
                    }
                }
      
                stage('Run vFile tests on ESX 6.0') {
                    steps {
                        node (VDVS_60_NODE_ID as String) { 
                            script{
                                try {
                                    echo "Build, deploy, and test vFile on 6.0 setup"
                                    echo "Build vFile binaries"
                                    echo "ESX=$ESX, VM1=$VM1, VM2=$VM2, VM3=$VM3;" 
                                    sh "make build-vfile-all" 
                                    echo " Deploy the vFile binaries"
                                    sh "make deploy-vfile-plugin"
                                    echo "Run the vFile tests"
                                    sh "make test-e2e-vfile"
                                    echo "vFile tests finished"
                                } finally {
                                    def cleanVfile = "True"
                                    cleanSetup(cleanVfile)   
                                } 
                            }
                        }
                    }
                }
            }
        }

        stage('Test Windows plugin') {
        /* This builds, deploys and tests the windows binaries */
            steps {
                node (VDVS_65_NODE_ID as String) { 
                    echo "Build, deploy, and test Windows plugin"
                    echo "Windows-VM=$WIN_VM1"
                    echo "Build Windows plugin binaries"
                    sh "make build-windows-plugin"
                    echo "Finished building binaries for windows"
                    echo "Deploy the Windows plugin binaries"
                    sh "make deploy-windows-plugin"
                    echo "echo Finished deploying binaries for windows"
                    echo "echo Run the Windows plugin tests"
                    sh "make test-e2e-windows"
                    echo "Windows plugin tests finished"
                }  
            }
        }
    }

    // The options directive is for configuration that applies to the whole job.
    options {
        // This ensures we only have 10 builds at a time, so
        // we don't fill up our storage!
        buildDiscarder(logRotator(numToKeepStr:'10'))
    }

    post {
        always {
            script {
                echo "Resetting nodes..."
                replaceNodeLabel(VDVS_65_NODE_NAME, VDVS_65_NODE_ID, "available")
                replaceNodeLabel(VDVS_60_NODE_NAME, VDVS_60_NODE_ID, "available")
                echo "Reset complete!"
            }
        }
    }
}

def replaceNodeLabel(nodeName, oldLabel, newLabel) {
    echo "Replacing label for node: " + nodeName + ", oldLabel: " + oldLabel + ", newLabel: " + newLabel
    def jk = jenkins.model.Jenkins.instance
    def node = jk.getNode(nodeName)
    node.labelString = node.labelString.replaceAll("\\b " + oldLabel + "\\b", "")
    node.labelString = node.labelString + " " + newLabel
    node.save()
}

def cleanSetup(cleanVfile) {
    echo "Cleaning up the setup..."
    echo "ESX = $ESX; VM1=$VM1; VM2=$VM2; VM3=$VM3;"
    echo "bool=$cleanVfile"
    def stopContainers = "docker stop \$(docker ps -a -q) 2> /dev/null || true"
    def removeContainers = "docker rm \$(docker ps -a -q) 2> /dev/null || true"
    def removeVolumes =  "docker volume rm \$(docker volume ls -q -f dangling=true) 2> /dev/null || true"
    sh "ssh ${env.GOVC_USERNAME}@$VM1 ${stopContainer}; ${removeContainer}; ${removeVolume}"
    sh "ssh ${env.GOVC_USERNAME}@$VM2 ${stopContainer}; ${removeContainer}; ${removeVolume}"
    sh "ssh ${env.GOVC_USERNAME}@$VM3 ${stopContainer}; ${removeContainer}; ${removeVolume}"
    if(cleanVfile.equals("True")){
        echo "Removing vFile plugin..."
        sh "make clean-vfile"
    }
    sh "make clean-all"
    echo "Cleanup finished."
}
