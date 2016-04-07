#! /usr/bin/env python

# Admin CLI for vmdk_opsd

import argparse

def main():
  namespace = parse_args()
  print namespace

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
  args = parser.parse_args()
  args.func(args)

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
  choices=['created-by', 'created', 'last-attached', 'datastore', 'policy', 'capacity',
      'used', 'attached-to']
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
  print "Called ls with args {0}".format(args)

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
