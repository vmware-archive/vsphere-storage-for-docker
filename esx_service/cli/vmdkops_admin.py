#!/usr/bin/env python
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

# Admin CLI for vmdk_opsd

import argparse
import os
import subprocess
import sys
import signal
import os.path
import shutil
import time

import vmdk_ops
# vmdkops python utils are in PY_LOC, so add to path.
sys.path.insert(0, vmdk_ops.PY_LOC)

import volume_kv as kv
import cli_table
import vsan_policy
import vmdk_utils
import vsan_info
import log_config
import auth
import auth_data_const
import convert
import auth_api
import auth_data
from auth_data import DB_REF

NOT_AVAILABLE = 'N/A'
UNSET = "Unset"

# Volume attributes
VOL_SIZE = 'size'
VOL_ALLOC = 'allocated'

def main():
    log_config.configure()
    kv.init()
    if not vmdk_ops.is_service_available():
       sys.exit('Unable to connect to the host-agent on this host, ensure the ESXi host agent is running before retrying.')
    args = parse_args()
    if args:
       args.func(args)


def commands():
    """
    This function returns a dictionary representation of a CLI specification that is used to
    generate a CLI parser. The dictionary is recursively walked in the `add_subparser()` function
    and appropriate calls are made to the `argparse` module to create a CLI parser that fits the
    specification.

    Each key in the top level of the dictionary is a command string. Each command may contain the
    following keys:

    * func - The callback function to be called when the command is issued. This key is always
             present unless there are subcommands, denoted by a 'cmds' key.

    * help - The help string that is printed when the `-h` or `--help` paramters are given without
             reference to a given command. (i.e. `./vmdkops_admin.py -h`). All top level help
             strings are printed in this instance.

    * args - A dictionary of any positional or optional arguments allowed for the given command. The
             args dictionary may contain the following keys:

             * help - The help for a given option which is displayed when the `-h` flag is given
                      with mention to a given command. (i.e. `./vmdkops_admin.py volume ls -h`). Help for
                      all options are shown for the command.

             * action - The action to take when the option is given. This is directly passed to
                        argparse. Note that `store_true` just means pass the option to the callback
                        as a boolean `True` value and don't require option parameters.
                        (i.e. `./vmdkops_admin.py volume ls -l`). Other options for the action value can be
                        found in the argparse documentation.
                        https://docs.python.org/3/library/argparse.html#action

             * metavar - A way to refer to each expected argument in help documentation. This is
                         directly passed to argparse.
                         See https://docs.python.org/3/library/argparse.html#metavar

             * required - Whether or not the argument is required. This is directly passed to
                          argparse.

             * type - A type conversion function that takes the option parameter and converts it
                      to a given type before passing it to the func callback. It prints an error and
                      exits if the given argument cannot be converted.
                      See https://docs.python.org/3/library/argparse.html#type

             * choices - A list of choices that can be provided for the given option. This list is
                         not directly passed to argparse. Instead a type conversion function is
                         created that only allows one or more of the choices as a comma separated
                         list to be supplied. An error identical to the one presented when using the
                         'choices' option in argparse is printed if an invalid choice is given. The
                         rationale for not directly using the argparse choices option is that
                         argparse requires space separated arguments of the form: `-l a b c`, rather
                         than the defacto single argument, comma separated form: `-l a,b,c`, common
                         to most unix programs.

    * cmds - A dictionary of subcommands where the key is the next word in the command line string.
             For example, in `vmdkops_admin.py tenant create`, `tenant` is the command, and `create` is
             the subcommand. Subcommands can have further subcommands, but currently there is only
             one level of subcommands in this specification. Each subcommand can contain the same
             attributes as top level commands: (func, help, args, cmds). These attributes have
             identical usage to the top-level keys, except they only apply when the subcommand is
             part of the command. For example the `--vm-list` argument only applies to `tenant
             create` or `tenant set` commands. It will be invalid in any other context.

             Note that the last subcommand in a chain is the one where the callback function is
             defined. For example, `tenant create` has a callback, but if a user runs the program
             like: `./vmdkops_admin.py tenant` they will get the following error:
             ```
             usage: vmdkops_admin.py tenant [-h] {rm,create,volume,get} ...
             vmdkops_admin.py tenant: error: too few arguments
             ```
    """

    return {
        'volume' : {
            'help': "Manipulate volumes",
            'cmds': {
                'ls': {
                    'func': ls,
                    'help': 'List volumes',
                    'args': {
                        '-c': {
                            'help': 'Display selected columns',
                            'choices': ['volume', 'datastore', 'vmgroup', 'capacity', 'used',
                                        'fstype', 'policy', 'disk-format', 'attached-to', 'access',
                                        'attach-as', 'created-by', 'created'],
                            'metavar': 'Col1,Col2,...'
                        },
                        '--vmgroup' : {
                            'help': 'Displays volumes for a given vmgroup'
                        }
                    }
                },
                'set': {
                    'func': set_vol_opts,
                    'help': 'Edit settings for a given volume',
                    'args': {
                        '--volume': {
                            'help': 'Volume to set options for, specified as "volume@datastore".',
                            'required': True
                        },
                        '--vmgroup': {
                            'help': 'Name of the vmgroup the volume belongs to.',
                            'required': True
                        },
                        '--options': {
                            'help': 'Options (specifically, access) to be set on the volume.',
                            'required': True
                        }
                    }
                }
            }
        },
        'policy': {
            'help': 'Configure and display storage policy information',
            'cmds': {
                'create': {
                    'func': policy_create,
                    'help': 'Create a storage policy',
                    'args': {
                        '--name': {
                            'help': 'The name of the policy',
                            'required': True
                        },
                        '--content': {
                            'help': 'The VSAN policy string',
                            'required': True
                        }
                    }
                },
                'rm': {
                    'func': policy_rm,
                    'help': 'Remove a storage policy',
                    'args': {
                        '--name': {
                            'help': 'Policy name',
                            'required': True
                        }
                    }
                },
                'ls': {
                    'func': policy_ls,
                    'help':
                    'List storage policies and volumes using those policies'
                },
                'update': {
                    'func': policy_update,
                    'help': ('Update the definition of a storage policy and all'
                              'VSAN objects using that policy'),
                    'args': {
                        '--name': {
                            'help': 'The name of the policy',
                            'required': True
                        },
                        '--content': {
                            'help': 'The VSAN policy string',
                            'required': True
                        }
                    }
                }
            }
        },
        'vmgroup': {
            #
            # vmgroup {create, update, rm , ls} - manipulates vmgroup
            # vmgroup vm {add, rm, ls}  - manipulates VMs for a vmgroup
            # vmgroup access {add, set, rm, ls} - manipulates datastore access right for a vmgroup
            #
            # Internally, "vmgroup" is called "tenant".
            # We decided to keep the name of functions as "tenant_*" for now
            'help': 'Administer and monitor volume access control',
            'cmds': {
                'create': {
                    'func': tenant_create,
                    'help': 'Create a new vmgroup',
                    'args': {
                        '--name': {
                            'help': 'The name of the vmgroup',
                            'required': True
                        },
                        '--description': {
                            'help': 'The description of the vmgroup',
                        },
                        # a shortcut allowing to add VMs on vmgroup Create
                        '--vm-list': {
                            'help': 'A list of VM names to place in this vmgroup',
                            'metavar': 'vm1, vm2, ...',
                            'type': comma_separated_string
                        }
                    }
                },
                'update': {
                    'func': tenant_update,
                    'help': 'Update an existing vmgroup',
                    'args': {
                        '--name': {
                            'help': 'The name of the vmgroup',
                            'required': True
                        },
                        '--new-name': {
                            'help': 'The new name of the vmgroup',
                        },
                        '--description': {
                            'help': 'The new description of the vmgroup',
                        },
                        '--default-datastore': {
                            'help': 'Datastore to be used by default for volumes placement',
                        }
                    }
                },
                'rm': {
                    'func': tenant_rm,
                    'help': 'Delete a vmgroup',
                    'args': {
                        '--name': {
                            'help': 'The name of the vmgroup',
                            'required': True
                      },
                      '--remove-volumes': {
                        'help': 'BE CAREFUL: Removes this vmgroup volumes when removing a vmgroup',
                        'action': 'store_true'
                      }
                    }
                },
                'ls': {
                    'func': tenant_ls,
                    'help': 'List vmgroups and the VMs they are applied to'
                },
                'vm': {
                    'help': 'Add, removes and lists VMs in a vmgroup',
                    'cmds': {
                        'add': {
                            'help': 'Add a VM(s)  to a vmgroup',
                            'func': tenant_vm_add,
                            'args': {
                                '--name': {
                                    'help': "Vmgroup to add the VM to",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to add to this vmgroup",
                                    'type': comma_separated_string,
                                    'required': True
                                }
                            }
                        },

                        'rm': {
                            'help': 'Remove VM(s) from a vmgroup',
                            'func': tenant_vm_rm,
                            'args': {
                                '--name': {
                                    'help': "Vmgroup to remove the VM from",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to rm from this vmgroup",
                                    'type': comma_separated_string,
                                    'required': True
                                }
                            }
                        },

                        'replace': {
                            'help': 'Replace VM(s) for a vmgroup',
                            'func': tenant_vm_replace,
                            'args': {
                                '--name': {
                                    'help': "Vmgroup to replace the VM for",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to replace for this vmgroup",
                                    'type': comma_separated_string,
                                    'required': True
                                }
                            }
                        },

                        'ls': {
                            'help': "list VMs in a vmgroup",
                            'func': tenant_vm_ls,
                            'args': {
                                '--name': {
                                    'help': "Vmgroup to list the VMs for",
                                    'required': True
                                }
                            }
                        }
                    }
                },
                'access': {
                    'help': 'Add or remove Datastore access and quotas for a vmgroup',
                    'cmds': {
                        'add': {
                            'func': tenant_access_add,
                            'help': 'Add a datastore access for a vmgroup',
                            'args': {
                                '--name': {
                                    'help': 'The name of the vmgroup',
                                    'required': True
                                },
                                '--datastore': {
                                    'help': "Datastore which access is controlled",
                                    'required': True
                                },
                                '--default-datastore': {
                                    'help': "Mark datastore as a default datastore for this vmgroup",
                                    'action': 'store_true'
                                },
                                '--allow-create': {
                                    'help': 'Allow create and delete on datastore if set',
                                    'action': 'store_true'
                                },
                                '--volume-maxsize': {
                                    'help': 'Maximum size of the volume that can be created',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                },
                                '--volume-totalsize': {
                                    'help':
                                    'Maximum total size of all volume that can be created on the datastore for this vmgroup',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                }
                            }
                        },
                        'set': {
                            'func': tenant_access_set,
                            'help': 'Modify datastore access for a vmgroup',
                            'args': {
                                '--name': {
                                    'help': 'The name of the vmgroup',
                                    'required': True
                                },
                                '--datastore': {
                                    'help': "Datastore name",
                                    'required': True
                                },
                                '--allow-create': {
                                    'help':
                                    'Allow create and delete on datastore if set to True; disallow create and delete on datastore if set to False',
                                    'metavar': 'Value{True|False} - e.g. True'
                                },
                                '--volume-maxsize': {
                                    'help': 'Maximum size of the volume that can be created',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                },
                                '--volume-totalsize': {
                                    'help':
                                    'Maximum total size of all volume that can be created on the datastore for this vmgroup',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                }
                            }
                        },
                        'rm': {
                            'func': tenant_access_rm,
                            'help': "Remove all access to a datastore for a vmgroup",
                            'args': {
                                '--name': {
                                    'help': 'The name of the vmgroup',
                                    'required': True
                                },
                                '--datastore': {
                                    'help': "Datstore which access is controlled",
                                    'required': True
                                }
                            }
                        },
                        'ls': {
                            'func': tenant_access_ls,
                            'help': 'List all access info for a vmgroup',
                            'args': {
                                '--name': {
                                    'help': 'The name of the vmgroup',
                                    'required': True
                                }
                            }
                        }
                    }
                }
            }
        },
        'config': {
            'help': 'Init and manage Config DB which enables quotas and access control',
            'cmds': {
                'init': {
                    'func': config_init,
                    'help': 'Init ' + DB_REF + ' to allows quotas and access groups (vm-groups)',
                    'args': {
                        '--datastore': {
                            'help': DB_REF + ' will be placed on a shared datastore',
                        },
                        '--local': {
                            'help': 'Allows local (SingleNode) Init',
                            'action': 'store_true'
                        },
                        '--force': {
                            'help': 'Force operation, ignore warnings',
                            'action': 'store_true'
                        }
                    }
                },
                'rm': {
                    'func': config_rm,
                    'help': 'Remove ' + DB_REF,
                    'args': {
                        '--local': {
                            'help': 'Remove only local link or local DB',
                            'action': 'store_true'
                        },
                        '--no-backup': {
                            'help': 'Do not create DB backup before removing',
                            'action': 'store_true'
                        },
                        '--confirm': {
                            'help': 'Explicitly confirm the operation',
                            'action': 'store_true'
                        }
                    }

                },
                'mv': {
                    'func': config_mv,
                    'help': 'Relocate ' + DB_REF + ' from its current location [not supported yet]',
                    'args': {
                        '--force': {
                            'help': 'Force operation, ignore warnings',
                            'action': 'store_true'
                        },
                        '--to': {
                            'help': 'Where to move the DB to.',
                            'required': True
                        }
                    }
                },
                'status': {
                    'func': config_status,
                    'help': 'Show the status of the Config DB'
                }
            }
        },
        'status': {
            'func': status,
            'help': 'Show the status of the vmdk_ops service',
            'args': {
                '--fast': {
                    'help': 'Skip some of the data collection (port, version)',
                    'action': 'store_true'
                }
            }
        }
    }


def create_parser():
    """ Create a CLI parser via argparse based on the dictionary returned from commands() """
    parser = argparse.ArgumentParser(description='vSphere Docker Volume Service admin CLI')
    add_subparser(parser, commands(), title='Manage VMDK-based Volumes for Docker')
    return parser


def add_subparser(parser, cmds_dict, title="", description=""):
    """ Recursively add subcommand parsers based on a dictionary of commands """
    subparsers = parser.add_subparsers(title=title, description=description, help="action")
    for cmd, attributes in cmds_dict.items():
        subparser = subparsers.add_parser(cmd, help=attributes['help'])
        if 'func' in attributes:
            subparser.set_defaults(func=attributes['func'])
        if 'args' in attributes:
            for arg, opts in attributes['args'].items():
                opts = build_argparse_opts(opts)
                subparser.add_argument(arg, **opts)
        if 'cmds' in attributes:
            add_subparser(subparser, attributes['cmds'], title=attributes['help'])


def build_argparse_opts(opts):
    if 'choices' in opts:
        opts['type'] = make_list_of_values(opts['choices'])
        help_opts = opts['help']
        opts['help'] = '{0}: Choices = {1}'.format(help_opts, opts['choices'])
        del opts['choices']
    return opts


def parse_args():
    parser = create_parser()
    args = parser.parse_args()
    if args != argparse.Namespace():
       return args
    else:
       parser.print_help()


def comma_separated_string(string):
    return string.split(',')


def make_list_of_values(allowed):
    """
    Take a list of allowed values for an option and return a function that can be
    used to typecheck a string of given values and ensure they match the allowed
    values.  This is required to support options that take comma separated lists
    such as --rights in 'tenant set --rights=create,delete,mount'
    """

    def list_of_values(string):
        given = string.split(',')
        for g in given:
            if g not in allowed:
                msg = (
                    'invalid choices: {0} (choices must be a comma separated list of '
                    'only the following words \n {1}. '
                    'No spaces are allowed between choices.)').format(g, repr(allowed).replace(' ', ''))
                raise argparse.ArgumentTypeError(msg)
        return given

    return list_of_values


def ls(args):
    """
    Print a table of all volumes and their datastores when called with no args.
    If args.l is True then show all metadata in a table.
    If args.c is not empty only display columns given in args.c (implies -l).
    """
    tenant_reg = '*'
    if args.vmgroup:
        tenant_reg = args.vmgroup

    if args.c:
        (header, rows) = ls_dash_c(args.c, tenant_reg)
    else:
        header = all_ls_headers()
        rows = generate_ls_rows(tenant_reg)
    print(cli_table.create(header, rows))


def ls_dash_c(columns, tenant_reg):
    """ Return only the columns requested in the format required for table construction """
    all_headers = all_ls_headers()
    all_rows = generate_ls_rows(tenant_reg)
    indexes = []
    headers = []
    choices = commands()['volume']['cmds']['ls']['args']['-c']['choices']
    for i, choice in enumerate(choices):
        if choice in columns:
            indexes.append(i)
            headers.append(all_headers[i])
    rows = []
    for row in all_rows:
        rows.append([row[i] for i in indexes])
    return (headers, rows)


def all_ls_headers():
    """ Return a list of all header for ls -l """
    return ['Volume', 'Datastore', 'VMGroup', 'Capacity', 'Used', 'Filesystem', 'Policy',
            'Disk Format', 'Attached-to', 'Access', 'Attach-as', 'Created By', 'Created Date']

def generate_ls_rows(tenant_reg):
    """ Gather all volume metadata into rows that can be used to format a table """
    rows = []
    for v in vmdk_utils.get_volumes(tenant_reg):
        if 'tenant' not in v or v['tenant'] == auth_data_const.ORPHAN_TENANT:
            tenant = 'N/A'
        else:
            tenant = v['tenant']
        path = os.path.join(v['path'], v['filename'])
        name = vmdk_utils.strip_vmdk_extension(v['filename'])
        metadata = get_metadata(path)
        attached_to = get_attached_to(metadata)
        policy = get_policy(metadata, path)
        size_info = get_vmdk_size_info(path)
        created, created_by = get_creation_info(metadata)
        diskformat = get_diskformat(metadata)
        fstype = get_fstype(metadata)
        access = get_access(metadata)
        attach_as = get_attach_as(metadata)
        rows.append([name, v['datastore'], tenant, size_info['capacity'], size_info['used'], fstype, policy,
                     diskformat, attached_to, access, attach_as, created_by, created])

    return rows


def get_creation_info(metadata):
    """
    Return the creation time and creation vm for a volume given its metadata
    """
    try:
        return (metadata[kv.CREATED], metadata[kv.CREATED_BY])
    except:
        return (NOT_AVAILABLE, NOT_AVAILABLE)


def get_attached_to(metadata):
    """ Return which VM a volume is attached to based on its metadata. """
    try:
        if kv.ATTACHED_VM_UUID in metadata:
            vm_name = vmdk_ops.vm_uuid2name(metadata[kv.ATTACHED_VM_UUID])
            if vm_name:
                return vm_name
            # If vm name couldn't be retrieved through uuid, use name from KV
            elif kv.ATTACHED_VM_NAME in metadata:
                return metadata[kv.ATTACHED_VM_NAME]
            else:
                return metadata[kv.ATTACHED_VM_UUID]
        else:
            return kv.DETACHED
    except:
        return kv.DETACHED

def get_attach_as(metadata):
    """ Return which mode a volume is attached as based on its metadata """
    try:
        return metadata[kv.VOL_OPTS][kv.ATTACH_AS]
    except:
        return kv.DEFAULT_ATTACH_AS


def get_access(metadata):
    """ Return the access mode of a volume based on its metadata """
    try:
       return metadata[kv.VOL_OPTS][kv.ACCESS]
    except:
        return kv.DEFAULT_ACCESS

def get_policy(metadata, path):
    """ Return the policy for a volume given its volume options """
    try:
        return metadata[kv.VOL_OPTS][kv.VSAN_POLICY_NAME]
    except:
        pass

    if vsan_info.is_on_vsan(path):
        return kv.DEFAULT_VSAN_POLICY
    else:
        return NOT_AVAILABLE

def get_diskformat(metadata):
    """ Return the Disk Format of the volume based on its metadata """
    try:
        return metadata[kv.VOL_OPTS][kv.DISK_ALLOCATION_FORMAT]
    except:
        return NOT_AVAILABLE

def get_fstype(metadata):
    """ Return the Filesystem Type of the volume based on its metadata """
    try:
        return metadata[kv.VOL_OPTS][kv.FILESYSTEM_TYPE]
    except:
        return NOT_AVAILABLE

def get_metadata(volPath):
    """ Take the absolute path to volume vmdk and return its metadata as a dict """
    return kv.getAll(volPath)


def get_vmdk_size_info(path):
    """
    Get the capacity and used space for a given VMDK given its absolute path.
    Values are returned as strings in human readable form (e.g. 10MB)

    Using get_vol_info api from volume kv. The info returned by this
    api is in human readable form
    """
    try:
        vol_info = kv.get_vol_info(path)
        if not vol_info: # race: volume is already gone
            return {'capacity': NOT_AVAILABLE,
                    'used': NOT_AVAILABLE}
        return {'capacity': vol_info[VOL_SIZE],
                'used': vol_info[VOL_ALLOC]}
    except subprocess.CalledProcessError:
        sys.exit("Failed to retrieve volume info for {0}.".format(path) \
            + " VMDK corrupted. Please remove and then retry")


KB = 1024
MB = 1024*KB
GB = 1024*MB
TB = 1024*GB
def human_readable(size_in_bytes):
    """
    Take an integer size in bytes and convert it to MB, GB, or TB depending
    upon size.
    """
    if size_in_bytes >= TB:
        return '{:.2f}TB'.format(size_in_bytes/TB)
    if size_in_bytes >= GB:
        return '{:.2f}GB'.format(size_in_bytes/GB)
    if size_in_bytes >= MB:
        return '{:.2f}MB'.format(size_in_bytes/MB)
    if size_in_bytes >= KB:
        return '{:.2f}KB'.format(size_in_bytes/KB)

    return '{0}B'.format(size_in_bytes)


def policy_create(args):
    output = vsan_policy.create(args.name, args.content)
    if output:
        print(output)
    else:
        print('Successfully created policy: {0}'.format(args.name))


def policy_rm(args):
    output = vsan_policy.delete(args.name)
    if output:
        print(output)
    else:
        print('Successfully removed policy: {0}'.format(args.name))


def policy_ls(args):
    volumes = vsan_policy.list_volumes_and_policies()
    policies = vsan_policy.get_policies()
    header = ['Policy Name', 'Policy Content', 'Active']
    rows = []
    used_policies = {}
    for v in volumes:
        policy_name = v['policy']
        if policy_name in used_policies:
            used_policies[policy_name] = used_policies[policy_name] + 1
        else:
            used_policies[policy_name] = 1

    for name, content in policies.items():
        if name in used_policies:
            active = 'In use by {0} volumes'.format(used_policies[name])
        else:
            active = 'Unused'
        rows.append([name, content.strip(), active])

    print(cli_table.create(header, rows))


def policy_update(args):
    output = vsan_policy.update(args.name,  args.content)
    if output:
        print(output)
    else:
        print('Successfully updated policy: {0}'.format(args.name))


def status(args):
    """Prints misc. status information. Returns an array of 1 element dicts"""
    result = []
    # version is extracted from localcli... slow...
    result.append({"=== Service": ""})
    version = "?" if args.fast else str(get_version())
    result.append({"Version": version})
    (service_status, pid) = get_service_status()
    result.append({"Status": str(service_status)})
    if pid:
        result.append({"Pid": str(pid)})
        port = "?" if args.fast else str(get_listening_port(pid))
        result.append({"Port": port})
    result.append({"LogConfigFile": log_config.LOG_CONFIG_FILE})
    result.append({"LogFile": log_config.LOG_FILE})
    result.append({"LogLevel": log_config.get_log_level()})
    result.append({"=== Authorization Config DB": ""})
    result += config_db_get_status()
    for r in result:
        print("{}: {}".format(list(r.keys())[0], list(r.values())[0]))

    return None


def set_vol_opts(args):
    try:
        set_ok = vmdk_ops.set_vol_opts(args.volume, args.vmgroup, args.options)
        if set_ok:
            print('Successfully updated settings for : {0}'.format(args.volume))
        else:
            print('Failed to update {0} for {1}.'.format(args.options, args.volume))
    except Exception as ex:
        print('Failed to update {0} for {1} - {2}.'.format(args.options,
                                                           args.volume,
                                                           str(ex)))


VMDK_OPSD = '/etc/init.d/vmdk-opsd'
PS = 'ps -c | grep '
GREP_V_GREP = ' | grep -v grep'
NOT_RUNNING_STATUS = ("Stopped", None)

def get_service_status():
    """
    Determine whether the service is running and it's PID. Return the 2 tuple
    containing a status string and PID. If the service is not running, PID is
    None
    """
    try:
        output = subprocess.check_output([VMDK_OPSD, "status"]).split()
        if output[2] == "not":
            return NOT_RUNNING_STATUS

        pidstr = output[3]
        pidstr = pidstr.decode('utf-8')
        pid = pidstr.split("=")[1]
        return ("Running", pid)
    except subprocess.CalledProcessError:
        return NOT_RUNNING_STATUS


def get_listening_port(pid):
    """ Return the configured port that the service is listening on """
    try:
        cmd = "{0}{1}{2}".format(PS, pid, GREP_V_GREP)
        output = subprocess.check_output(cmd, shell=True).split()[6]
        return output.decode('utf-8')
    except:
        return NOT_AVAILABLE

def get_version():
    """ Return the version of the installed VIB """
    try:
        cmd = 'localcli software vib list | grep esx-vmdkops-service'
        version_str = subprocess.check_output(cmd, shell=True).split()[1]
        return version_str.decode('utf-8')
    except:
        return NOT_AVAILABLE

def operation_fail(error_info):
    print(error_info)
    return error_info

def tenant_ls_headers():
    """ Return column names for tenant ls command """
    headers = ['Uuid', 'Name', 'Description', 'Default_datastore', 'VM_list']
    return headers

def generate_vm_list(vms_uuid):
    """ Generate vm names with given list of vm uuid"""
    # vms_uuid is a list of vm_uuid
    # example: vms_uuid=["vm1_uuid", "vm2_uuid"]
    # the return value is a string like this vm1,vm2
    res = ""
    for vm_uuid in vms_uuid:
        vm_name = vmdk_utils.get_vm_name_by_uuid(vm_uuid)
        # If the VM name cannot be resolved then its possible
        # the VM has been deleted or migrated off the host,
        # skip the VM in that case.
        if vm_name:
            res = res + vm_name
            res = res + ","

    if res:
        res = res[:-1]

    return res

def generate_tenant_ls_rows(tenant_list):
    """ Generate output for tenant ls command """
    rows = []
    for tenant in tenant_list:
        uuid = tenant.id
        name = tenant.name
        description = tenant.description
        if not tenant.default_datastore_url or tenant.default_datastore_url == auth_data_const.DEFAULT_DS_URL:
            default_datastore = ""
        else:
            default_datastore = vmdk_utils.get_datastore_name(tenant.default_datastore_url)

        vm_list = generate_vm_list(tenant.vms)
        rows.append([uuid, name, description, default_datastore, vm_list])

    return rows

def tenant_create(args):
    """ Handle tenant create command """
    error_info, _ = auth_api._tenant_create(
        name=args.name,
        description="",
        vm_list=args.vm_list,
        privileges=[])
    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup '{}' is created. Do not forget to run 'vmgroup vm add' and "
              "'vmgroup access add' to enable access control.".format(args.name))

def tenant_update(args):
    """ Handle tenant update command """
    error_info = auth_api._tenant_update(name=args.name,
                                         new_name=args.new_name,
                                         description=args.description,
                                         default_datastore=args.default_datastore)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup modify succeeded")

def tenant_rm(args):
    """ Handle tenant rm command """
    remove_volumes = False
    # If args "remove_volumes" is not specified in CLI
    # args.remove_volumes will be None
    if args.remove_volumes:
        print("All Volumes will be removed")
        remove_volumes = True

    error_info = auth_api._tenant_rm(args.name, remove_volumes)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup rm succeeded")

def tenant_ls(args):
    """ Handle tenant ls command """
    error_info, tenant_list = auth_api._tenant_ls()
    if error_info:
        return operation_fail(error_info.msg)

    header = tenant_ls_headers()
    rows = generate_tenant_ls_rows(tenant_list)
    print(cli_table.create(header, rows))

def tenant_vm_add(args):
    """ Handle tenant vm add command """
    error_info = auth_api._tenant_vm_add(args.name, args.vm_list)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup vm add succeeded")

def tenant_vm_rm(args):
    """ Handle tenant vm rm command """
    error_info = auth_api._tenant_vm_rm(args.name, args.vm_list)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup vm rm succeeded")

def tenant_vm_replace(args):
    """ Handle tenant vm replace command """
    error_info = auth_api._tenant_vm_replace(args.name, args.vm_list)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup vm replace succeeded")

def tenant_vm_ls_headers():
    """ Return column names for tenant vm ls command """
    headers = ['Uuid', 'Name']
    return headers

def generate_tenant_vm_ls_rows(vms):
    """ Generate output for tenant vm ls command """
    rows = []
    for vm in vms:
        # vm has the format like this (vm_uuid)
        uuid = vm
        name = vmdk_utils.get_vm_name_by_uuid(uuid)
        rows.append([uuid, name])

    return rows

def tenant_vm_ls(args):
    """ Handle tenant vm ls command """

    # Handling _DEFAULT tenant case separately to print info message
    # instead of printing empty list
    if (args.name == auth_data_const.DEFAULT_TENANT):
        print("{0} tenant contains all VMs which were not added to other tenants".format(auth_data_const.DEFAULT_TENANT))
        return

    error_info, vms = auth_api._tenant_vm_ls(args.name)
    if error_info:
        return operation_fail(error_info.msg)

    header = tenant_vm_ls_headers()
    rows = generate_tenant_vm_ls_rows(vms)
    print(cli_table.create(header, rows))


def tenant_access_add(args):
    """ Handle tenant access command """
    volume_maxsize_in_MB = None
    volume_totalsize_in_MB = None
    if args.volume_maxsize:
        volume_maxsize_in_MB = convert.convert_to_MB(args.volume_maxsize)
    if args.volume_totalsize:
        volume_totalsize_in_MB = convert.convert_to_MB(args.volume_totalsize)

    error_info = auth_api._tenant_access_add(name=args.name,
                                             datastore=args.datastore,
                                             default_datastore=args.default_datastore,
                                             allow_create=args.allow_create,
                                             volume_maxsize_in_MB=volume_maxsize_in_MB,
                                             volume_totalsize_in_MB=volume_totalsize_in_MB
                                            )

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup access add succeeded")

def tenant_access_set(args):
    """ Handle tenant access set command """
    volume_maxsize_in_MB = None
    volume_totalsize_in_MB = None
    if args.volume_maxsize:
        volume_maxsize_in_MB = convert.convert_to_MB(args.volume_maxsize)
    if args.volume_totalsize:
        volume_totalsize_in_MB = convert.convert_to_MB(args.volume_totalsize)

    error_info = auth_api._tenant_access_set(name=args.name,
                                             datastore=args.datastore,
                                             allow_create=args.allow_create,
                                             volume_maxsize_in_MB=volume_maxsize_in_MB,
                                             volume_totalsize_in_MB=volume_totalsize_in_MB)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup access set succeeded")

def tenant_access_rm(args):
    """ Handle tenant access rm command """
    error_info = auth_api._tenant_access_rm(args.name, args.datastore)
    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("vmgroup access rm succeeded")

def tenant_access_ls_headers():
    """ Return column names for tenant access ls command """
    headers = ['Datastore', 'Allow_create', 'Max_volume_size', 'Total_size']
    return headers

def generate_tenant_access_ls_rows(privileges):
    """ Generate output for tenant access ls command """
    rows = []
    for p in privileges:
        if not p.datastore_url or p.datastore_url == auth_data_const.DEFAULT_DS_URL:
            datastore = ""
        else:
            datastore = vmdk_utils.get_datastore_name(p.datastore_url)
        allow_create = ("False", "True")[p.allow_create]
        # p[auth_data_const.COL_MAX_VOLUME_SIZE] is max_volume_size in MB
        max_vol_size = UNSET if p.max_volume_size == 0 else human_readable(p.max_volume_size * MB)
        # p[auth_data_const.COL_USAGE_QUOTA] is total_size in MB
        total_size = UNSET if p.usage_quota == 0 else human_readable(p.usage_quota * MB)
        rows.append([datastore, allow_create, max_vol_size, total_size])

    return rows

def tenant_access_ls(args):
    """ Handle tenant access ls command """
    name = args.name
    error_info, privileges = auth_api._tenant_access_ls(name)
    if error_info:
        return operation_fail(error_info.msg)

    header = tenant_access_ls_headers()
    rows = generate_tenant_access_ls_rows(privileges)
    print(cli_table.create(header, rows))

# ==== CONFIG DB manipulation functions ====

def create_db_symlink(path, link_path):
    """Force-creates a symlink to path"""
    if os.path.islink(link_path):
        os.remove(link_path)
    try:
        os.symlink(path, link_path)
    except Exception as ex:
        print("Failed to create symlink at {} to {}".format(link_path, path))
        sys.exit(ex)


def db_move_to_backup(path):
    """
    Saves a DB copy side by side. Basically, glorified copy to a unique file name.
    Returns target name
    """
    target = "{}.bak_{}".format(path, time.asctime().replace(" ", "_"))
    # since we generate unique file name, no need to check if it exists
    shutil.move(path, target)
    return target


def is_local_vmfs(datastore_name):
    """return True if datastore is local VMFS one"""
    # TODO - check for datastore being on local VMFS volume.

    # the code below is supposed to do it, but in ESX 6.5 it returns
    # " local = <unset>", so leaving it out for now
    # def vol_info_from_vim(datastore_name):
    #     si = pyVim.connect.Connect()
    #     host = pyVim.host.GetHostSystem(si)
    #     fss = host.configManager.storageSystem.fileSystemVolumeInfo.mountInfo
    #     vmfs_volume_info = [f.volume for f in fss if f.volume.name == datastore_name and
    #                         f.volume.type == "VMFS"]
    #     return vmfs_volume_info and vmfs_volume_info.local

    return False


def service_reset():
    """Send a signal to the service to restart itself"""
    (status, pid) = get_service_status()
    if pid:
        os.kill(int(pid), signal.SIGUSR1)
    return None


def err_out(_msg, _info=None):
    """A helper to print a message with (optional) info about DB MOde. Returns the message"""
    print(_msg)
    if _info:
        print("Additional information: {}".format(_info))
    return _msg


def err_override(_msg, _info):
    """A helper to print messates with extra help about --force flag"""
    new_msg = "Error: {}".format(_msg) + " . Add '--force' flag to force the request execution"
    return err_out(new_msg, _info)


def config_elsewhere(datastore):
    """Returns a list of config DBs info on other datastore, or empty list"""
    # Actual implementation: scan vim datastores, check for dockvols/file_name
    #  return None or list of (db_name, full_path) tuples for existing config DBs.
    others = []
    for (ds_name, _, dockvol_path) in vmdk_utils.get_datastores():
        full_path = os.path.join(dockvol_path, auth_data.CONFIG_DB_NAME)
        if ds_name != datastore and os.path.exists(full_path):
            others.append((ds_name, full_path))
    return others


def check_ds_local_args(args):
    """
    checks consistency in --local and --datastore args, an datastore presense
    :Return: None for success, errmsg for error
    """
    if args.datastore:
        ds_name = args.datastore
        if not os.path.exists(os.path.join("/vmfs/volumes", ds_name)):
            return err_out("No such datastore: {}".format(ds_name))
    if args.datastore and args.local:
        return err_out("Error: only one of '--datastore' or '--local' can be set")
    if not args.datastore and not args.local:
        return err_out("Error: one of '--datastore' or '--local' have to be set")
    return None


def config_init(args):
    """
    Init Config DB to allows quotas and access groups (vm-groups)
    :return: None for success, string for error
    """

    err = check_ds_local_args(args)
    if err:
        return err

    if args.datastore:
        ds_name = args.datastore
        db_path = auth_data.AuthorizationDataManager.ds_to_db_path(ds_name)
    else:
        db_path = auth_data.AUTH_DB_PATH

    link_path = auth_data.AUTH_DB_PATH # where was the DB, now is a link

    # Check the existing config mode
    with auth_data.AuthorizationDataManager() as auth:
        try:
            auth.connect()
            info = auth.get_info()
            mode = auth.mode # for usage outside of the 'with'
        except auth_data.DbAccessError as ex:
            return err_out(str(ex))

    if mode == auth_data.DBMode.NotConfigured:
        pass
    elif mode == auth_data.DBMode.MultiNode or mode == auth_data.DBMode.SingleNode:
        return err_out(DB_REF + " is already initialized. Use 'rm --local' to reset", info)
    else:
        raise Exception("Fatal: Internal error - unknown mode: {}".format(mode))

    if args.datastore:
        # Check that the target datastore is NOT local VMFS, bail out if it is (--force to overide).
        if is_local_vmfs(ds_name) and not args.force:
            return err_override("{} is a local datastore.".format(ds_name) +
                                "Shared datastores are recommended.", "N/A")
        # Check other datastores, bail out if dockvols/DB exists there.
        other_ds_config = config_elsewhere(ds_name)
        if len(other_ds_config) > 0  and not args.force:
            return err_override("Found " + DB_REF + "on other datastores.",
                                other_ds_config)

    if not os.path.exists(db_path):
        print("Creating new DB at {}".format(db_path))
        auth = auth_data.AuthorizationDataManager(db_path)
        err = auth.new_db()
        if err:
            return err_out("Init failed: %s" % str(err))

    # Almost done -  just create link and refresh the service
    if not args.local:
        print("Creating a symlink to {} at {}".format(db_path, link_path))
        create_db_symlink(db_path, link_path)

    return service_reset()


def config_rm(args):
    """
    Remove Local Config DB or local link. We NEVER remove shared DB.
    :return: None for success, string for error
    """

    # This asks for double confirmation, and removes the local link or DB (if any)
    # NEVER deletes the shared database - instead prints help

    if not args.local:
        return err_out("""
        Shared DB removal is not supported. For removing  local configuration, use --local flag.
        For removing shared DB,  run 'vmdkops_admin config rm --local' on ESX hosts using this DB,
        and manually remove the {} file from shared storage.
        """.format(auth_data.CONFIG_DB_NAME))

    if not args.confirm:
        return err_out("Warning: For extra safety, removal operation requires '--confirm' flag.")

    link_path = auth_data.AUTH_DB_PATH # local DB or link
    if not os.path.lexists(link_path):
        return None

    try:
        if not os.path.islink(link_path) and not args.no_backup:
            print("Moved {} to backup file {}".format(link_path,
                                                       db_move_to_backup(link_path)))
        else:
            os.remove(link_path)
            print("Removed link {}".format(link_path))
    except Exception as ex:
        print(" Failed to remove {}: {}".format(link_path, ex))

    return service_reset()


def config_mv(args):
    """[Not Supported Yet]
    Relocate config DB from its current location
    :return: None for success, string for error
    """

    if not args.force:
        return err_out(DB_REF + " move to {} ".format(args.to) +
                       "requires '--force' flag to execute the request.")

    # TODO:
    # this is pure convenience code, so it  is very low priority; still, here are the steps:
    # checks if target exists upfront, and fail if it does
    # cp the DB instance 'to' , and flip the symlink.
    # refresh service (not really needed as next vmci_command handlers will pick it up)
    # need --dryrun or --confirm
    # issue: works really with discovery only , as others need to find it out

    print("Sorry, configuration move ('config mv' command) is not supported yet")
    return None


def config_db_get_status():
    '''A helper fot get config DB status. Returns an array of status info'''
    result = []
    with auth_data.AuthorizationDataManager() as auth:
        try:
            auth.connect()
        except:
            pass # connect() will set the status regardess of success
        for (k, v) in auth.get_info().items():
            result.append({k: v})
    return result


def config_status(args):
    """A subset of 'config' command - prints the DB config only"""
    for r in config_db_get_status():
        print("{}: {}".format(list(r.keys())[0], list(r.values())[0]))
    return None


# ==== Run it now ====

if __name__ == "__main__":
    main()
