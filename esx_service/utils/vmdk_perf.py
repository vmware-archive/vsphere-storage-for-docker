# Copyright 2016 VMware, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

##
## Implements the interface for side car based implementation
## of a KV store for vmdks.
##

from ctypes import *
from vmware import vsi
import pyVim
import pyVim
from pyVim.connect import Connect, Disconnect
from pyVim import vmconfig
from pyVmomi import VmomiSupport, vim, vmodl
import sys
import logging
import atexit

# Format to return stats
FORMAT_CSV = vim.PerfFormat.csv
DEFAULT_PERF_FORMAT = FORMAT_CSV

# IO stats returned for the volume
IO_STATS = 'iostats'

# Lable per metric
LABEL = 'label'

# Summary description per metrix
SUMMARY = 'summary'

# Performance counters per vm:device
COUNTERS='counters'

# Default collection interval
COLLECTION_INTERVAL = 20

# Samples to retrieve
SAMPLE_SIZE = 1
DEFAULT_SAMPLE_SIZE = SAMPLE_SIZE

# Global map of counter IDs supported by the performance manager.
perf_counters_map = {}

# Global map of VMs and devices and performance counter IDs
vm_dev_map = {}

# Performance manager from the service instance
perfm = None

# The local service instance used by this module
si = None

def init_svc_inst():
    '''
    connect and do stuff on local machine
    '''
    global si
    global perfm

    # Connect to localhost as dcui
    # User "dcui" is a local Admin that does not lose permissions
    # when the host is in lockdown mode.
    si = pyVim.connect.Connect(host='localhost', user='dcui')
    if not si:
        raise SystemExit("Failed to connect to localhost as 'dcui'.")

    atexit.register(pyVim.connect.Disconnect, si)

    # set out ID in context to be used in request - so we'll see it in logs
    reqCtx = VmomiSupport.GetRequestContext()
    reqCtx["realUser"] = 'dvolplug'

    service_content = si.RetrieveContent()
    perfm = service_content.perfManager
    return si


def init_perf():
   global perfm
   global perf_counters_map
   
   init_svc_inst()

   # Create a map of performance counters that the performance
   # manager supports
   perf_counters = perfm.perfCounter
   for counter in perf_counters:
      perf_counters_map[counter.key] = {'label': counter.nameInfo.label, 'summary': counter.nameInfo.summary}
   return True

def init_perf_for_vol(vm, bus, unit):
   # Create the map for the given VM and the device at given
   # bus and unit number.
   vm_name = vm.config.name
   vm_uuid = vm.config.uuid
   try:
      vm_metrics = perfm.QueryAvailablePerfMetric(vm)
   except vim.fault.NotAuthenticated as ex:
      logging.exception(ex)
      init_svc_inst()
      nvm = si.content.searchIndex.FindByUuid(None, vm_uuid, True, False)
      vm_metrics = perfm.QueryAvailablePerfMetric(nvm)

   device = "scsi{0}:{1}".format(bus, unit)

   if not vm_name in vm_dev_map:
      vm_dev_map[vm_name] = {}

   if not device in vm_dev_map[vm_name]:
      vm_dev_map[vm_name][device] = {}

   counters = []
   for i in vm_metrics:
      if device == i.instance:
         counters += [i.counterId]
   vm_dev_map[vm_name][device] = {COUNTERS: counters}

def delete_perf_for_vol(vm, bus, unit):
   # Remove the metrics that were created earlier for this volume
   vm_name = vm.config.name
   device = "scsi{0}:{1}".format(bus, unit)
   if vm_name in vm_dev_map:
      vm_dev_map[vm_name][device] = None
   
def get_vol_stats(vm, bus, unit):
   # If the VM or the device was never initialized in the VM device map.
   vm_name = vm.config.name
   vm_uuid = vm.config.uuid
   device = "scsi{0}:{1}".format(bus, unit)

   if not vm_name in vm_dev_map or \
      not device in vm_dev_map[vm_name] or \
      not COUNTERS in vm_dev_map[vm_name][device]:
     return None

   metric_ids = []
   for counter in vm_dev_map[vm_name][device][COUNTERS]:
      metric_ids += [vim.PerfMetricId(counterId=counter, instance=device)]

   pspec = vim.PerfQuerySpec(entity=vm, format=DEFAULT_PERF_FORMAT, \
                             intervalId=COLLECTION_INTERVAL, \
                             maxSample=DEFAULT_SAMPLE_SIZE, \
                             metricId=metric_ids)

   try:
      metrics = perfm.QueryPerf([pspec])
   except vim.fault.NotAuthenticated as ex:
      logging.exception(ex)
      init_svc_inst()
      nvm = si.content.searchIndex.FindByUuid(None, vm_uuid, True, False)
      pspec = vim.PerfQuerySpec(entity=nvm, format=DEFAULT_PERF_FORMAT, \
                                intervalId=COLLECTION_INTERVAL, \
                                maxSample=DEFAULT_SAMPLE_SIZE, \
                                metricId=metric_ids)
      metrics = perfm.QueryPerf([pspec])

   if not metrics or metrics == []:
      logging.warning("No metrics retrieved.");
      return None

   # Build the response with label, summary and value for each counter
   perf_stats = {}
   print(metrics)
   for metric in metrics[0].value:
      label = perf_counters_map[metric.id.counterId][LABEL]
      summary = perf_counters_map[metric.id.counterId][SUMMARY]
      perf_stats[label] = {'value': metric.value, 'summary': summary}

   return perf_stats

