#! /usr/bin/env python

# This is the admin cli for esx vmdk volume management

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
  return parser

def parse_args():
  parser = create_parser()
  args = parser.parse_args()
  args.func(args)

def add_ls_parser(subparsers):
  parser = subparsers.add_parser('ls', help='List volumes')
  parser.add_argument('-l', action='store_true', help='List detailed information about volumes')
  choices=['created-by', 'created', 'last-attached', 'datastore', 'policy', 'capacity',
      'used', 'attached-to']
  parser.add_argument('-c', nargs='+', help='Only display given columns', choices=choices)
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

if __name__ == "__main__":
  main()
