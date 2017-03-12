#!/usr/bin/env python
#
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
# limitations under the License.#

# Logging config support.
# Allows customers to edit log config, which is a json file (see LOG_CONFIG_FILE).
# Also introduces log rotation.

import logging
import logging.config
import json
import os
import os.path

# Logging configuration is read from this file.
# If the file does not exist, we create it from from LOG_CONFIG_DEFAULT dictionary.
# Note: the file handler name MUST be "rotate_file",
# since we rely on it to locate log file name after config is loaded.
LOG_CONFIG_FILE = "/etc/vmware/vmdkops/log_config.json"

LOG_LEVEL_DEFAULT = 'DEBUG'

# Defaults for log files - used to generate conf file if it is missing
# Note: log file location should be synced with CI and 'make'
LOG_FILE = "/var/log/vmware/vmdk_ops.log"
LOG_MAX_BYTES = 1048576  # 1MB
LOG_MAX_BACKUPS = 1
LOG_CONFIG_DEFAULT = {
    "info": [
        "Logging configuration for vmdk_opsd service, in python logging config format.",
        "",
        "'level' defines verbosity and could be DEBUG, INFO, WARNING, ERROR, CRITICAL",
        "'maxBytes' and 'backupCount' define max log size and number of log backup files kept",
        "For more, see https://docs.python.org/2/library/logging.config.html#logging-config-dictschema",
        "",
        "Do NOT change 'rotate_file' name in handlers - it is used in code to locate the log file."
    ],
    "version":
    1,  # mandated by https://docs.python.org/2/library/logging.config.html#logging-config-dictschema
    "disable_existing_loggers": False,
    "formatters": {
        "standard": {
            "format":
            "%(asctime)-12s %(process)d [%(threadName)s] [%(levelname)-7s] %(message)s",
            "datefmt": "%x %X",
        }
    },
    "handlers": {
        "rotate_file": {
            "class": "logging.handlers.RotatingFileHandler",
            "filename": LOG_FILE,
            "formatter": "standard",
            "level": LOG_LEVEL_DEFAULT,
            "maxBytes": LOG_MAX_BYTES,
            "backupCount": LOG_MAX_BACKUPS,
            "encoding": "utf8",
        }
    },
    "loggers": {
        "": {
            "handlers": ["rotate_file"],
            "level": LOG_LEVEL_DEFAULT,
        },
    }
}


def configure(config_file=LOG_CONFIG_FILE):
    """
    Checks if the json config file exists, and if it does not, creates it from
    hardcoded defaults. Then loads log config from the  file.
    Returns log file name.
    In case of format mistakes in config file, this function
    simply throws an exception - since there is no log yet, we can't log the error.
    """

    generatedConf = False
    # if the conf file is not there, create it:
    if not os.path.isfile(config_file):
        try:
            os.makedirs(os.path.dirname(config_file))
        except:
            pass
        with open(config_file, 'w') as f:
            json.dump(LOG_CONFIG_DEFAULT, f, sort_keys=False, indent=4)
        generatedConf = True

    # Get the configuration info - now it *has* to be there
    with open(config_file) as f:
        conf = json.load(f)

    # make sure the dir for logs exists
    log = conf['handlers']['rotate_file']['filename']
    dir = os.path.dirname(log)
    if not os.path.isdir(dir):
        os.makedirs(dir)

    # Now configure logging
    logging.config.dictConfig(conf)
    if generatedConf:
        logging.info("Log configuration generated - '%s'." % config_file)
    return log


def get_log_level(config_file=LOG_CONFIG_FILE):
    """ Return the configured log level """
    try:
        with open(config_file) as f:
            conf = json.load(f)
        return conf['loggers']['']['level']
    except:
        # The log config file doesn't currently exist. Use the default.
        return LOG_LEVEL_DEFAULT

def get_logger(name, level=LOG_LEVEL_DEFAULT):
    """ Returns the logger with required log level set
    level defines verbosity and could be DEBUG, INFO, WARNING, ERROR, CRITICAL """

    logger = logging.getLogger(name)
    logger.setLevel(logging.getLevelName(level))
    return logger


# manual test: "sudo python log_config.py"
if __name__ == "__main__":
    conf_file = os.path.basename(LOG_CONFIG_FILE)
    log_file = configure(conf_file)
    print("logging to %s:\n" % log_file)
    logging.info("==== start log ====")
    logging.error("Trying an error !")
    logging.warning("still a warning")
    logging.info("OK info")
    logging.debug("debugging")
    # visual check. Caveat: prints ALL log file
    print("==========Content of %s========" % log_file)
    with open(log_file) as f:
        print(f.read())
    print("==========Content of %s========" % conf_file)
    with open(conf_file) as f:
        print(f.read())
