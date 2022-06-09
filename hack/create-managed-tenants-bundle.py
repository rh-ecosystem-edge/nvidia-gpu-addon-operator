#!/usr/bin/env python3

import os
import shutil
import yaml
import pathlib
from argparse import ArgumentParser

ANNOTATION_PATH = "metadata/annotations.yaml"
ADDON_PATH = "addons/nvidia-gpu-addon/main"
MANIFESTS = "manifests"
ADDON_NAME = "nvidia-gpu-addon-operator"
CSV_SUFFIX = "clusterserviceversion.yaml"



def get_csv_file(bundle_path):
    manifests = os.path.join(bundle_path, MANIFESTS)
    for file in os.listdir(manifests):
        if file.endswith(CSV_SUFFIX):
            return os.path.join(manifests, file)


def handle_csv(bundle_path, version, prev_version, channel):
    print("Handling csv")
    csv_file = get_csv_file(bundle_path)
    with open(csv_file, "r") as _f:
        csv = yaml.safe_load(_f)

    csv["metadata"]["annotations"]["olm.skipRange"] = f">=0.0.1 <{version}"
    csv["metadata"]["name"] = f"{ADDON_NAME}.{version}"
    csv["spec"]["version"] = f"{version}"
    csv["spec"]["maturity"] = f"{channel}"
    if prev_version != "null":
        csv["spec"]["replaces"] = f"{ADDON_NAME}.{prev_version}"
    with open(csv_file, "w") as _f:
        yaml.dump(csv, _f)

def handle_annotations(bundle_path, channel, namespace):
    print("Handling annotations")
    with open(os.path.join(bundle_path, ANNOTATION_PATH), "r") as _f:
        annotations = yaml.safe_load(_f)
    annotations["annotations"]["operators.operatorframework.io.bundle.channels.v1"] = channel
    annotations["annotations"]["operators.operatorframework.io.bundle.channel.default.v1"] = channel
    annotations["annotations"]["operators.operatorframework.io.bundle.package.v1"] = ADDON_NAME
    annotations["annotations"]["operatorframework.io/suggested-namespace"] = namespace
    with open(os.path.join(bundle_path, ANNOTATION_PATH), "w") as _f:
        yaml.dump(annotations, _f)


def create_new_bundle(args):
    version = args.version
    addon_path = os.path.join(args.manage_tenants_bundle_path, ADDON_PATH)
    bundle_path = os.path.join(addon_path, version)
    if os.path.isdir(bundle_path):
        replace = False
        replace = input(f"Bundle version {version} already exists, Do you with to override [y/N]? ").lower()
        if replace == 'y':
            print(f"Replacing version {version} with new bundle...")
            shutil.rmtree(bundle_path)
        else:
            print(f"ERROR: bundle version ({version}) already exists. Path: {bundle_path}")
            exit(1)
    shutil.copytree("./bundle", bundle_path)
    shutil.rmtree(os.path.join(bundle_path, "tests"))
    #copy_tree("./bundle/", bundle_path)
    #remove_tree(os.path.join(bundle_path, "tests"))

    handle_annotations(bundle_path, args.channel, args.namespace)
    handle_csv(bundle_path, version, args.prev_version, args.channel)


if __name__ == '__main__':
    parser = ArgumentParser(
        __file__,
        description='adding new bundle version to nvidia-gpu-addon manage-tenants-bundles'
    )
    parser.add_argument(
        '-mP', '--manage-tenants-bundle-path',
        required=True,
        help='Path to managed tenants repo on the disk'
    )
    parser.add_argument(
        '-c', '--channel',
        default="alpha",
        required=False,
        help='Channel of addon'
    )
    parser.add_argument(
        '-n', '--namespace',
        default="redhat-nvidia-gpu-addon",
        help='Target namespace'
    )
    parser.add_argument(
        '-v', '--version',
        required=True,
        help='New addon version'
    )
    parser.add_argument(
        '-pv', '--prev-version',
        required=False,
        default="",
        help='Previous addon version'
    )

    args = parser.parse_args()
    create_new_bundle(args)
