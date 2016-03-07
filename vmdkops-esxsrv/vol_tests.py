import sys, getopt
import subprocess
import vmci_srv as vmci
import volumeKVStore as kv

# Default volumes dir
vmName = "testVM"
vols = ['vol1', 'vol2', 'vol3', 'vol4', 'vol5', 'vol6', 'vol7', 'vol8', 'vol9', 'vol10']
volopts = "size:1gb"

def doCreate(volDir):
   print "Creating volumes"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      vmci.createVMDK(volPath, vol, volopts)

   print "Verifying volume metadata"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      volDict = kv.getAll(volPath)

      print "Vol metadata 'status' - %s, 'volOpts' - %s" % (volDict['status'], volDict['volOpts'])
      if volDict['status'] != 'detached':
         print 'Found volume %s with status %s, expected' % (vol, volDict['status'], 'detached')

   return

def doAttach(volDir, vmName):
   print "Attaching volumes"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      vmci.attachVMDK(volPath, vmName)
   print "Verifying volume metadata"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      volDict = kv.getAll(volPath)

      if volDict['status'] != 'attached':
         print 'Found volume %s with status %s, expected' % (vol, volDict['status'], 'attached')

   return

def doDetach(volDir, vmName):
   print "Detaching volumes"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      vmci.detachVMDK(volPath, vmName)
   print "Verifying volume metadata"
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      volDict = kv.getAll(volPath)

      if volDict['status'] != 'detached':
         print 'Found volume %s with status %s, expected' % (vol, volDict['status'], 'detached')

   return

def doVolDelete(volDir):
   print 'Removing volumes'
   for vol in vols:
      volPath = os.path.join(volDir, "%s.vmdk" % vol)
      vmci.removeVMDK(volPath)
   return

def main(argv):
   if argv == []:
      print 'vol_tests.py -v <VM config path> -d <volumes dir>'
      sys.exit(2)

   try:
      opts, args = getopt.getopt(argv,"hv:d:")
   except getopt.GetoptError:
      print 'vol_tests.py -v <VM config path> -d <volumes dir>'
      sys.exit(2)

   for opt, arg in opts:
      if opt == '-h':
         print 'vol_tests.py -v <vm config path> -d <volumes dir>'
         sys.exit()
      elif opt in ("-v"):
         vmPath = arg
      elif opt in ("-d"):
         volDir = arg

   cmd = 'vim-cmd solo/registervm %s %s' % (vmPath, vmName)
   proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, shell=True)

   vmId = proc.communicate()[0]

   ret = proc.returncode

   if ret != 0:
      print "Failed to power on VM, exiting", vmPath
      sys.exit(0)

   # Start VM
   print "Starting VM %s with id %s ..." % (vmPath, vmId)

   cmd = 'vim-cmd vmsvc/power.on %s' % vmId
   subprocess.call(cmd, shell=True)

   # Create volumes
   doCreate(volDir)

   # Attach/Re-attach volumes
   doAttach(volDir, vmName)

   doAttach(volDir, vmName)

   # Check volume meta-data
   doVerifyVolMeta(volDir)

   # Detach volumes
   doDetach(volDir, vmName)

   # Delete volumes
   doVolDelete(volDir)

   cmd = 'vim-cmd vmsvc/power.off %s' % vmId
   subprocess.call(cmd, shell=True)

   cmd = 'vim-cmd vmsvc/unregister %s' % vmId
   subprocess.call(cmd, shell=True)

# start the server
if __name__ == "__main__":
    main(sys.argv[1:])
