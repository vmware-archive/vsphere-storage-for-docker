#!/bin/sh
cleanup() {
  make clean-vm VM=$VM1
  make clean-vm VM=$VM2
  make clean-esx ESX=$ESX
}
