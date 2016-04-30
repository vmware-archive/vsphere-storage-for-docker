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
import volumeKVStore as kv
import cli_table
import vmdk_ops

import pyVim.connect


def main():
    kv.init()
    args = parse_args()
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
             For example, in `vmdkops_admin.py role create`, `role` is the command, and `create` is
             the subcommand. Subcommands can have further subcommands, but currently there is only
             one level of subcommands in this specification. Each subcommand can contain the same
             attributes as top level commands: (func, help, args, cmds). These attributes have
             identical usage to the top-level keys, except they only apply when the subcommand is
             part of the command. For example the `--matches-vm` argument only applies to `role
             create` or `role set` commands. It will be invalid in any other context.

             Note that the last subcommand in a chain is the one where the callback function is
             defined. For example, `role create` has a callback, but if a user runs the program
             like: `./vmdkops_admin.py role` they will get the following error:
             ```
             usage: vmdkops_admin.py role [-h] {rm,create,set,ls,get} ...
             vmdkops_admin.py role: error: too few arguments
             ```
    """
    return {
        'ls': {
            'func': ls,
            'help': 'List volumes',
            'args': {
                '-l': {
                    'help': 'List detailed information about volumes',
                    'action': 'store_true',
                },
                '-c': {
                    'help': 'Display selected columns',
                    'choices': ['volume', 'datastore', 'created-by', 'created', 'last-attached',
                                'attached-to', 'policy', 'capacity', 'used'],
                    'metavar': 'Col1,Col2,...'
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
                    'help': 'List storage policies and volumes using those policies'
                }
            }
        },
        'role': {
            'help': 'Administer and monitor volume access control',
            'cmds': {
                'create': {
                    'func': role_create,
                    'help': 'Create a new role',
                    'args': {
                        '--name': {
                            'help': 'The name of the role',
                            'required': True
                        },
                        '--matches-vm': {
                            'help': 'Apply this role to VMS with names matching Glob',
                            'metavar': 'Glob1,Glob2,...',
                            'required': True,
                            'type': comma_seperated_string
                        },
                        '--rights': {
                            'help': 'Permissions granted to matching VMs',
                            'required': True,
                            'choices': ['create', 'delete', 'mount'],
                            'metavar': 'create,delete,mount'
                        },
                        '--volume-maxsize': {
                            'help': 'Maximum size of the volume that can be created',
                            'required': True,
                            'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                        }
                    }
                },
                'rm': {
                    'func': role_rm,
                    'help': 'Delete a role',
                    'args': {
                        'name': {
                            'help': 'The name of the role'
                        }
                    }
                },
                'ls': {
                    'func': role_ls,
                    'help': 'List roles and the VMs they are applied to'
                },
                'set': {
                    'func': role_set,
                    'help': 'Modify an existing role',
                    'args': {
                        '--name': {
                            'help': 'The name of the role',
                            'required': True
                        },
                        '--matches-vm': {
                            'help': 'Apply this role to VMS with names matching Glob',
                            'metavar': 'Glob1,Glob2,...',
                            'type': comma_seperated_string
                        },
                        '--rights': {
                            'help': 'Permissions granted to matching VMs',
                            'choices': ['create', 'delete', 'mount'],
                            'metavar': 'create,delete,mount'
                        },
                        '--volume-maxsize': {
                            'help': 'Maximum size of the volume that can be created',
                            'metavar': 'Num{MB,GB,TB} - e.g. 2TB'
                        }
                    }
                },
                'get': {
                    'func': role_get,
                    'help': 'Get all roles and permissions for a given VM',
                    'args': {
                        'vm_name': {
                            'help': 'The name of the VM'
                        }
                    }
                }
            }
        },
        'status': {
            'func': status,
            'help': 'Show the status of the vmdk_ops service'
        }
    }

def create_parser():
    """ Create a CLI parser via argparse based on the dictionary returned from commands() """
    parser = argparse.ArgumentParser(description='Manage VMDK Volumes')
    add_subparser(parser, commands())
    return parser

def add_subparser(parser, commands):
    """ Recursively add subcommand parsers based on a dictionary of commands """
    subparsers = parser.add_subparsers()
    for cmd, attributes in commands.iteritems():
        subparser = subparsers.add_parser(cmd, help=attributes['help'])
        if 'func' in attributes:
            subparser.set_defaults(func=attributes['func'])
        if 'args' in attributes:
            for arg, opts in attributes['args'].iteritems():
                opts = build_argparse_opts(opts)
                subparser.add_argument(arg, **opts)
        if 'cmds' in attributes:
            add_subparser(subparser, attributes['cmds'])

def build_argparse_opts(opts):
    if 'choices' in opts:
        opts['type'] = make_list_of_values(opts['choices'])
        help = opts['help']
        opts['help'] = '{0}: Choices = {1}'.format(help, opts['choices'])
        del opts['choices']
    return opts

def parse_args():
    parser = create_parser()
    return parser.parse_args()

def comma_seperated_string(string):
    return string.split(',')

def make_list_of_values(allowed):
    """
    Take a list of allowed values for an option and return a function that can be
    used to typecheck a string of given values and ensure they match the allowed
    values.  This is required to support options that take comma seperated lists
    such as --rights in 'role set --rights=create,delete,mount'
    """
    def list_of_values(string):
        given = string.split(',')
        for g in given:
            if g not in allowed:
                msg = "invalid choice: {0} (choose from {1})".format(g, allowed)
                raise argparse.ArgumentTypeError(msg)
        return given
    return list_of_values

def ls(args):
    """
    Print a table of all volumes and their datastores when called with no args.
    If args.l is True then show all metadata in a table.
    If args.c is not empty only display columns given in args.c (implies -l).
    """
    if args.c:
        (header, data) = ls_dash_c(args.c)
    elif args.l:
        (header, data) = ls_dash_l()
    else:
        (header, data) = ls_no_args()
    print cli_table.create(header, data)

def ls_no_args():
    """
    Collect all volume names and their datastores as lists,
    stripping the '.vmdk' from the volume name
    """
    header = ['Volume', 'Datastore']
    data = [[vmdk_ops.strip_vmdk_extension(v['filename']), v['datastore']] for v in get_volumes()]
    return (header, data)

def ls_dash_l():
    """
    List all volumes and relevant metadata
    """
    header = all_ls_headers()
    rows = generate_ls_dash_l_rows()
    return (header, rows)

def ls_dash_c(columns):
    """ Return only the columns requested in the format required for table construction """
    all_headers = all_ls_headers()
    all_rows = generate_ls_dash_l_rows()
    indexes = []
    headers = []
    for i in range(len(all_headers)):
        if format_header_as_arg(all_headers[i]) in columns:
            indexes.append(i)
            headers.append(all_headers[i])
    rows = []
    for row in all_rows:
        rows.append([row[i] for i in indexes])
    return (headers, rows)

def format_header_as_arg(header):
    """
    Take a header formatted as words seperated by spaces starting with
    capitals and convert it to a cli argument friendly format that is all
    lowercase with words seperated by dashes. i.e. 'Created By' -> 'created-by'
    """
    return '-'.join(header.lower().split())

def all_ls_headers():
    """ Return a list of all header for ls -l """
    return ['Volume', 'Datastore', 'Created By', 'Created', 'Last Attached', 'Attached To',
            'Policy', 'Capacity', 'Used']

def generate_ls_dash_l_rows():
   """ Gather all volume metadata into rows that can be used to format a table """
   rows = []
   for v in get_volumes():
       path = os.path.join(v['path'], v['filename'])
       name = vmdk_ops.strip_vmdk_extension(v['filename'])
       metadata = get_metadata(path)
       if metadata[u'status'] == u'attached':
           attached_to = metadata[u'attachedVMUuid']
       else:
           attached_to = 'detached'
       size_info = get_vmdk_size_info(path)
       rows.append([name, v['datastore'], 'N/A', 'N/A', 'N/A', attached_to, 'N/A',
                    size_info['capacity'], size_info['used']])
   return rows

def is_symlink(path):
    """ Returns true if the file is a symlink """
    try:
        os.readlink(path)
        return True
    except OSError:
        return False

# datastores should not change during 'vmdkops_admin' run,
# so using global to avoid multiple scans of /vmfs/volumes
datastores = None

def get_datastores():
    """
    Return pairs of datastore names and absolute paths to dockvols directory,
    after following the symlink
    """

    global datastores
    if datastores != None:
        return datastores

    si = pyVim.connect.Connect()
    #  We are connected to ESX so chileEntity[0] is current DC/Host
    ds_objects = si.content.rootFolder.childEntity[0].datastoreFolder.childEntity
    datastores = [(d.info.name, os.path.join(d.info.url, 'dockvols'))
                  for d in ds_objects]
    pyVim.connect.Disconnect(si)

    return datastores

def get_volumes():
    """ Return dicts of docker volumes, their datastore and their paths """
    volumes = []
    for (datastore, path) in get_datastores():
        for file in list_vmdks(path):
            volumes.append({'path': path, 'filename': file, 'datastore': datastore})
    return volumes

def get_metadata(volPath):
    """ Take the absolute path to volume vmdk and return its metadata as a dict """
    return kv.getAll(volPath)

def get_vmdk_size_info(path):
    """
    Get the capacity and used space for a given VMDK given its absolute path
    Currently this data is retrieved via a call to vmkfstools.
    The output being parsed looks like the following:

    Capacity bytes: 209715200
    Used bytes: 27262976
    Unshared bytes: 27262976
    """
    try:
        cmd = "vmkfstools --extendedstatinfo {0}".format(path).split()
        output = subprocess.check_output(cmd)
        lines = output.split('\n')
        return {'capacity': lines[0].split()[2], 'used': lines[1].split()[2]}
    except CalledProcessError:
        sys.exit("Failed to stat {0}. VMDK corrupted. Please remove and then retry".format(path))

def list_vmdks(path):
    """ Return a list all VMDKs in a given path """
    try:
        files = os.listdir(path)
        return [f for f in files if vmdk_ops.vmdk_is_a_descriptor(os.path.join(path, f))]
    except OSError as e:
        # dockvols may not exists on a datastore, so skip it
        return []

def policy_create(args):
    print "Called policy_create with args {0}".format(args)

def policy_rm(args):
    print "Called policy_rm with args {0}".format(args)

def policy_ls(args):
    print "Called policy_ls with args {0}".format(args)

def role_create(args):
    print "Called role_create with args {0}".format(args)

def role_rm(args):
    print "Called role_rm with args {0}".format(args)

def role_ls(args):
    print "Called role_ls with args {0}".format(args)

def role_set(args):
    print "Called role_set with args {0}".format(args)

def role_get(args):
    print "Called role_get with args {0}".format(args)

def status(args):
    print "Called status with args {0}".format(args)

if __name__ == "__main__":
    main()
