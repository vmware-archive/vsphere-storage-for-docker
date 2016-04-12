#! /usr/bin/env python
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
import volumeKVStore as kv
import cli_table

volume_path = "/vmfs/volumes"

def main():
    kv.init()
    args = parse_args()
    args.func(args)

def create_parser():
    parser = argparse.ArgumentParser(description='Manage VMDK Volumes')
    subparsers = parser.add_subparsers();
    add_ls_parser(subparsers)
    add_df_parser(subparsers)
    add_policy_parsers(subparsers)
    add_role_parsers(subparsers)
    add_status_parser(subparsers)
    return parser

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

def add_ls_parser(subparsers):
    parser = subparsers.add_parser('ls', help='List volumes')
    parser.add_argument('-l', action='store_true', help='List detailed information about volumes')
    choices = [format_header_as_arg(h) for h in all_ls_headers()]
    parser.add_argument('-c', help='Only display given columns: Choices = {0}'.format(choices),
      type=make_list_of_values(choices), metavar='Column1,Column2,...')
    parser.set_defaults(func=ls)

def add_df_parser(subparsers):
    parser = subparsers.add_parser('df', help='Show datastore usage and availability')
    parser.set_defaults(func=df)

def add_policy_parsers(subparsers):
    parser = subparsers.add_parser('policy', help='Configure and display storage policy information')
    policy_subparsers = parser.add_subparsers();
    add_policy_create_parser(policy_subparsers)
    add_policy_rm_parser(policy_subparsers)
    add_policy_ls_parser(policy_subparsers)

def add_policy_create_parser(subparsers):
    parser = subparsers.add_parser('create', help='Create a storage policy')
    parser.add_argument('--name', help='The name of the policy', required=True)
    parser.add_argument('--content', help='The VSAN policy string', required=True)
    parser.set_defaults(func=policy_create)

def add_policy_rm_parser(subparsers):
    parser = subparsers.add_parser('rm', help='Remove a storage policy')
    parser.add_argument('name', help='Policy name')
    parser.set_defaults(func=policy_rm)

def add_policy_ls_parser(subparsers):
    parser = subparsers.add_parser('ls',
      help='List storage policies and volumes using those policies')
    parser.set_defaults(func=policy_ls)

def add_role_parsers(subparsers):
    parser = subparsers.add_parser('role', help='Administer and monitor volume access control')
    role_subparsers = parser.add_subparsers()
    add_role_create_parser(role_subparsers)
    add_role_rm_parser(role_subparsers)
    add_role_ls_parser(role_subparsers)
    add_role_set_parser(role_subparsers)
    add_role_get_parser(role_subparsers)

def add_role_create_parser(subparsers):
    parser = subparsers.add_parser('create', help='Create a new role')
    parser.add_argument('--name', help='The name of the role', required=True)
    parser.add_argument('--matches-vm', metavar='Glob1,Glob2,...', required=True,
      help='Apply this role to VMs with names matching Glob', type=comma_seperated_string)
    parser.add_argument('--rights', help='Permissions granted to matching VMs', required=True,
      type=make_list_of_values(['create', 'delete', 'mount']), metavar='create,delete,mount')
    parser.add_argument('--volume-maxsize', help='Maximum size of the volume that can be created',
      required=True, metavar='Num{MB,GB,TB} - e.g. 2TB')
    parser.set_defaults(func=role_create)

def add_role_rm_parser(subparsers):
    parser = subparsers.add_parser('rm', help='Delete a role')
    parser.add_argument('name', help='The name of the role')
    parser.set_defaults(func=role_rm)

def add_role_ls_parser(subparsers):
    parser = subparsers.add_parser('ls', help='List roles and the VMs they are applied to')
    parser.set_defaults(func=role_ls)

def add_role_set_parser(subparsers):
    parser = subparsers.add_parser('set', help='Modify an existing role')
    parser.add_argument('--name', help='The name of the role', required=True)
    parser.add_argument('--matches-vm', nargs="*",
      help='Apply this role to VMs with names matching Glob', metavar='Glob')
    parser.add_argument('--rights', help='Permissions granted to matching VMs', nargs="*",
      choices=['create', 'delete', 'mount'])
    parser.add_argument('--volume-maxsize', help='Maximum size of the volume that can be created',
      metavar='Num{MB,GB,TB} - e.g. 2TB')
    parser.set_defaults(func=role_set)

def add_role_get_parser(subparsers):
    parser = subparsers.add_parser('get', help='Get all roles and permissions for a given VM')
    parser.add_argument('vm_name', help='The name of the VM')
    parser.set_defaults(func=role_get)

def add_status_parser(subparsers):
    parser = subparsers.add_parser('status', help='Show the status of vmdk_ops service')
    parser.set_defaults(func=status)

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
    data = [[v[1][0:-5], v[2]] for v in get_volumes()]
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
   volumes = get_volumes()
   for v in get_volumes():
       path = os.path.join(v[0], v[1])
       name = v[1][0:-5] # strip .vmdk
       datastore = v[2]
       metadata = get_metadata(path)
       if metadata[u'status'] == u'attached':
           attached_to = metadata[u'attachedVMUuid']
       else:
           attached_to = 'detached'
       capacity = metadata[u'volOpts'][u'size']
       rows.append([name, datastore, 'N/A', 'N/A', 'N/A', attached_to, 'N/A', capacity, 'N/A'])
   return rows

def is_symlink(path):
    """ Returns true if the file is a symlink """
    try:
        os.readlink(path)
        return True
    except OSError:
        return False

def get_datastores():
    """ Return pairs of datastore names and their absolute paths after following the symlink """
    datastores = []
    for f in os.listdir(volume_path):
        ds_path = os.path.join(volume_path, f)
        if is_symlink(ds_path):
            datastores.append((f, os.path.join(volume_path, os.readlink(ds_path), 'dockvols')))
    return datastores

def get_volumes():
    """ Return tuples of volumes, their datastore and their paths """
    volumes = []
    for (datastore, path) in get_datastores():
        files = os.listdir(path)
        for f in files:
            if f.endswith('.vmdk') and not f.endswith('-flat.vmdk'):
                volumes.append((path, f, datastore))
    return volumes

def get_metadata(volPath):
    """ Take the absolute path to volume vmdk and return its metadata as a dict """
    return kv.getAll(volPath)

def df(args):
    print "Called df with args {0}".format(args)

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
