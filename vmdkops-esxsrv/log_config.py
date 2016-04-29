#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Log config support

import logging
import logging.config
import json
import os
import os.path


# Logging configuration is read from this this file
# Notice that the file handled SHOULD be named "rotate_file", 
# since we rely on it to locate log file name after config is loaded.
# Also, since we install via VIB, the actual file will be immutable on ESX,
# so to edit it on ESX, rename to .bck, then copy .bck to log_config.json and edit.
LOG_CONFIG_FILE="/etc/vmware/vmdkops/log_config.json"

# Defaults for log files - used ONLY when config file read failed 
# Note: log file location should be synced with CI and 'make'
LOG_FILE = "/var/log/vmware/vmdk_ops.log"
LOG_MAX_BYTES   = 10485760 # 10MB
LOG_MAX_BACKUPS = 5
LOG_FALLBACK_CONFIG = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "standard": {
            "format": "%(asctime)-12s %(process)d [%(levelname)-7s] %(message)s",
            "datefmt": "%x %X",
        }
    },
    "handlers": {
        "rotate_file": {
            "class": "logging.handlers.RotatingFileHandler",
            "filename": LOG_FILE,
            "formatter": "standard",
            "level": "INFO",
            "maxBytes": LOG_MAX_BYTES,
            "backupCount": LOG_MAX_BACKUPS,
            "encoding": "utf8",
        }
    },
    "loggers": {
        "": {
            "handlers": ["rotate_file"],
            "level": "INFO",
        },
    }
}

def configure(config_file=LOG_CONFIG_FILE):
    """
    Loads log config from json file. If fails, uses hardcoded defaults.
    Returns log file name, bool indicating if default is use, and err msg (if any)
    """
    usingDefaults = True
    msg=""

    # Get the configuration info, from a file or hardcoded defaults:
    try:
        with open(config_file) as f:
            conf=json.load(f)
        usingDefaults = False
    except Exception as e:
        conf=LOG_FALLBACK_CONFIG
        msg=str(e)

    # make sure the dir for logs exists
    log=conf['handlers']['rotate_file']['filename']
    dir=os.path.dirname(log)
    if not os.path.isdir(dir):
        os.makedirs(dir)

    # Now configure logging
    logging.config.dictConfig(conf)
    if usingDefaults:
        logging.warning("Failed to parse logging configuration '%s'. %s. Using defaults."
                        % (config_file, msg))
    return log, usingDefaults, msg


# manual test: "sudo python log_config.py"
if __name__ == "__main__":
    log, usingDefaults, msg = configure("log_config.json")
    if usingDefaults:
        print("Failed to parse logging configuration '%s'. %s. Using defaults."
                        % (LOG_CONFIG_FILE, msg))
    print("logging to %s:\n" % log)
    logging.info("==== start log ====")
    logging.error("Trying an error !")
    logging.warning("still a warning")
    logging.info("OK info")
    logging.debug("debugging")
    # visual check. Caveat: prints ALL log file
    with open(log) as f:
        print(f.read())

