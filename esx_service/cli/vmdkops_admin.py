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
import auth_data
import auth_api

NOT_AVAILABLE = 'N/A'
UNSET = "Unset"

# Volume attributes
VOL_SIZE = 'size'
VOL_ALLOC = 'allocated'

def main():
    log_config.configure()
    kv.init()
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
                      with mention to a given command. (i.e. `./vmdkops_admin.py ls -h`). Help for
                      all options are shown for the command.

             * action - The action to take when the option is given. This is directly passed to
                        argparse. Note that `store_true` just means pass the option to the callback
                        as a boolean `True` value and don't require option parameters.
                        (i.e. `./vmdkops_admin.py ls -l`). Other options for the action value can be
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
             usage: vmdkops_admin.py tenant [-h] {rm,create,set,ls,get} ...
             vmdkops_admin.py tenant: error: too few arguments
             ```
    """
    return {
        'ls': {
            'func': ls,
            'help': 'List volumes',
            'args': {
                '-c': {
                    'help': 'Display selected columns',
                    'choices': ['volume', 'datastore', 'created-by', 'created',
                                'attached-to', 'policy', 'capacity', 'used','disk-format',
                                'fstype', 'access', 'attach-as'],
                    'metavar': 'Col1,Col2,...'
                },
                '--tenant' : {
                    'help': 'Displays volumes for a given tenant'
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
                        'name': {
                            'help': 'Policy name'
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
        #
        # tenant {create, update, rm , ls} - manipulates tenants
        # tenant vm {add, rm, ls}  - manipulates VMs for a tenant
        # tenant access {add, set, rm, ls} - manipulates datastore access right for a tenant
        #
        'tenant': {
            'help': 'Administer and monitor volume access control',
            'cmds': {
                'create': {
                    'func': tenant_create,
                    'help': 'Create a new tenant',
                    'args': {
                        '--name': {
                            'help': 'The name of the tenant',
                            'required': True
                        },
                        '--description': {
                            'help': 'The description of the tenant',
                        },
                        # a shortcut allowing to add VMs on Tenant Create
                        '--vm-list': {
                            'help': 'A list of VM names to place in this Tenant',
                            'metavar': 'vm1, vm2, ...',
                            'type': comma_seperated_string
                        }
                    }
                },
                'update': {
                    'func': tenant_update,
                    'help': 'Update an existing tenant',
                    'args': {
                        '--name': {
                            'help': 'The name of the tenant',
                            'required': True
                        },
                        '--new-name': {
                            'help': 'The new name of the tenant',
                        },
                        '--description': {
                            'help': 'The new description of the tenant',
                        },
                        '--default-datastore': {
                            'help': 'The name of the datastore to be used by default for volumes placement',
                        }
                    }
                },
                'rm': {
                    'func': tenant_rm,
                    'help': 'Delete a tenant',
                    'args': {
                        '--name': {
                            'help': 'The name of the tenant',
                            'required': True
                      },
                      '--remove-volumes': {
                        'help': 'BE CAREFUL: Removes this tenant volumes when removing a tenant',
                        'action': 'store_true'
                      }
                    }
                },
                'ls': {
                    'func': tenant_ls,
                    'help': 'List tenants and the VMs they are applied to'
                },
                'vm': {
                    'help': 'Add, removes and lists VMs in a tenant',
                    'cmds': {
                        'add': {
                            'help': 'Add a VM(s)  to a tenant',
                            'func': tenant_vm_add,
                            'args': {
                                '--name': {
                                    'help': "Tenant to add the VM to",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to add to this Tenant",
                                    'type': comma_seperated_string,
                                    'required': True
                                }
                            }
                        },

                        'rm': {
                            'help': 'Remove VM(s) from a tenant',
                            'func': tenant_vm_rm,
                            'args': {
                                '--name': {
                                    'help': "Tenant to remove the VM from",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to rm from this Tenant",
                                    'type': comma_seperated_string,
                                    'required': True
                                }
                            }
                        },

                        'replace': {
                            'help': 'Replace VM(s) for a tenant',
                            'func': tenant_vm_replace,
                            'args': {
                                '--name': {
                                    'help': "Tenant to replace the VM for",
                                    'required': True
                                },
                                '--vm-list': {
                                    'help': "A list of VM names to replace for this Tenant",
                                    'type': comma_seperated_string,
                                    'required': True
                                }
                            }
                        },

                        'ls': {
                            'help': "list VMs in a tenant",
                            'func': tenant_vm_ls,
                            'args': {
                                '--name': {
                                    'help': "Tenant to list the VMs for",
                                    'required': True
                                }
                            }
                        }
                    }
                },
                'access': {
                    'help': 'Add or remove Datastore access and quotas for a tenant',
                    'cmds': {
                        'add': {
                            'func': tenant_access_add,
                            'help': 'Add a datastore access for a tenant',
                            'args': {
                                '--name': {
                                    'help': 'The name of the tenant',
                                    'required': True
                                },
                                '--datastore': {
                                    'help': "Datastore which access is controlled",
                                    'required': True
                                },
                                '--default-datastore': {
                                    'help': "Mark datastore as a default datastore for this tenant",
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
                                    'Maximum total size of all volume that can be created on the datastore for this tenant',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                }
                            }
                        },
                        'set': {
                            'func': tenant_access_set,
                            'help': 'Modify datastore access for a tenant',
                            'args': {
                                '--name': {
                                    'help': 'Tenant name',
                                    'required': True
                                },
                                '--datastore': {
                                    'help': "Datastore name",
                                    'required': True
                                },
                                '--allow-create': {
                                    'help': 
                                    'Allow create and delete on datastore if set to True; disallow create and delete on datastore if set to False',
                                },
                                '--volume-maxsize': {
                                    'help': 'Maximum size of the volume that can be created',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                },
                                '--volume-totalsize': {
                                    'help':
                                    'Maximum total size of all volume that can be created on the datastore for this tenant',
                                    'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                                }
                            }
                        },
                        'rm': {
                            'func': tenant_access_rm,
                            'help': "Remove all access to a datastore for a tenant",
                            'args': {
                                '--name': {
                                    'help': 'The name of the tenant',
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
                            'help': 'List all access info for a tenant',
                            'args': {
                                '--name': {
                                    'help': 'The name of the tenant',
                                    'required': True
                                }
                            }
                        }
                    }
                }
            }
        },
        'status': {
            'func': status,
            'help': 'Show the status of the vmdk_ops service'
        },
        'set': {
            'func': set_vol_opts,
            'help': 'Edit settings for a given volume',
            'args': {
                '--volume': {
                    'help': 'Volume to set options for, specified as "volume@datastore".',
                    'required': True
                },
                '--options': {
                    'help': 'Options (specifically, access) to be set on the volume.',
                    'required': True
                }
            }
        }
    }


def create_parser():
    """ Create a CLI parser via argparse based on the dictionary returned from commands() """
    parser = argparse.ArgumentParser(description='Manage VMDK Volumes')
    add_subparser(parser, commands())
    return parser


def add_subparser(parser, cmds_dict):
    """ Recursively add subcommand parsers based on a dictionary of commands """
    subparsers = parser.add_subparsers()
    for cmd, attributes in cmds_dict.items():
        subparser = subparsers.add_parser(cmd, help=attributes['help'])
        if 'func' in attributes:
            subparser.set_defaults(func=attributes['func'])
        if 'args' in attributes:
            for arg, opts in attributes['args'].items():
                opts = build_argparse_opts(opts)
                subparser.add_argument(arg, **opts)
        if 'cmds' in attributes:
            add_subparser(subparser, attributes['cmds'])


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


def comma_seperated_string(string):
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
    if args.tenant:
        tenant_reg = args.tenant

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
    choices = commands()['ls']['args']['-c']['choices']
    for i in range(len(choices)):
        if choices[i] in columns:
            indexes.append(i)
            headers.append(all_headers[i])
    rows = []
    for row in all_rows:
        rows.append([row[i] for i in indexes])
    return (headers, rows)


def all_ls_headers():
    """ Return a list of all header for ls -l """
    return ['Volume', 'Datastore', 'Created By VM', 'Created',
            'Attached To VM', 'Policy', 'Capacity', 'Used',
            'Disk Format', 'Filesystem Type', 'Access', 'Attach As']

def generate_ls_rows(tenant_reg):
    """ Gather all volume metadata into rows that can be used to format a table """
    rows = []
    for v in vmdk_utils.get_volumes(tenant_reg):
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
        rows.append([name, v['datastore'], created_by, created, attached_to,
                     policy, size_info['capacity'], size_info['used'],
                     diskformat, fstype, access, attach_as])
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
    """ Return which VM a volume is attached to based on its metadata """
    try:
        vm_name = vmdk_ops.vm_uuid2name(metadata[kv.ATTACHED_VM_UUID])
        if not vm_name:
            return kv.DETACHED
        return vm_name
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
    print("Version: {0}".format(get_version()))
    (status, pid) = get_service_status()
    print("Status: {0}".format(status))
    if pid:
        print("Pid: {0}".format(pid))
        print("Port: {0}".format(get_listening_port(pid)))
    print("LogConfigFile: {0}".format(log_config.LOG_CONFIG_FILE))
    print("LogFile: {0}".format(log_config.LOG_FILE))
    print("LogLevel: {0}".format(log_config.get_log_level()))


def set_vol_opts(args):
    try:
        set_ok = vmdk_ops.set_vol_opts(args.volume, args.options)
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
        if not tenant.default_datastore_url or tenant.default_datastore_url == auth.DEFAULT_DS_URL:
            default_datastore = ""
        else:
            default_datastore = vmdk_utils.get_datastore_name(tenant.default_datastore_url)
        
        vm_list = generate_vm_list(tenant.vms)
        rows.append([uuid, name, description, default_datastore, vm_list])

    return rows

def tenant_create(args):
    """ Handle tenant create command """
    error_info, tenant = auth_api._tenant_create(
                                                 name=args.name, 
                                                 description="",  
                                                 vm_list=args.vm_list, 
                                                 privileges=[])
    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("tenant create succeeded")

def tenant_update(args):
    """ Handle tenant update command """
    error_info = auth_api._tenant_update(name=args.name,
                                         new_name=args.new_name,
                                         description=args.description,
                                         default_datastore=args.default_datastore)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("tenant modify succeeded")

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
        print("tenant rm succeeded")

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
        print("tenant vm add succeeded")

def tenant_vm_rm(args):
    """ Handle tenant vm rm command """
    error_info = auth_api._tenant_vm_rm(args.name, args.vm_list)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("tenant vm rm succeeded")

def tenant_vm_replace(args):
    """ Handle tenant vm replace command """
    error_info = auth_api._tenant_vm_replace(args.name, args.vm_list)

    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("tenant vm replace succeeded")

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
        print("tenant access add succeeded")

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
        print("tenant access set succeeded")

def tenant_access_rm(args):
    """ Handle tenant access rm command """
    error_info = auth_api._tenant_access_rm(args.name, args.datastore)
    if error_info:
        return operation_fail(error_info.msg)
    else:
        print("tenant access rm succeeded")

def tenant_access_ls_headers():
    """ Return column names for tenant access ls command """
    headers = ['Datastore', 'Allow_create', 'Max_volume_size', 'Total_size']
    return headers

def generate_tenant_access_ls_rows(privileges):
    """ Generate output for tenant access ls command """
    rows = []
    for p in privileges:
        if not p.datastore_url or p.datastore_url == auth.DEFAULT_DS_URL:
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

if __name__ == "__main__":
    main()
