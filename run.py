import json
import os
import subprocess

from typing import Literal

ignored_pair_dict: dict[str, Literal[True]] = {}

# Load ignored pairs from a file
if os.path.exists("ignored_images.txt"):
    with open("ignored_images.txt", "r") as f:
        ignored_pairs = f.readlines()
        ignored_pairs = [pair.strip() for pair in ignored_pairs if pair.strip()]
        for pair in ignored_pairs:
            p1, p2 = pair.split(":")
            ignored_pair_dict[pair] = True
            ignored_pair_dict["{}:{}".format(p2, p1)] = True

def compare_pair(image1: dict, image2: dict):
    if image1['width'] != image2['width'] or image1['height'] != image2['height']:
        return

    p1 = image1['path']
    p2 = image2['path']

    assert p1 != p2, "Image paths should not be the same"
    assert isinstance(p1, str) and isinstance(p2, str), "Image paths should be strings"

    if ignored_pair_dict.get("{}:{}".format(p1, p2), False) or ignored_pair_dict.get("{}:{}".format(p2, p1), False):
        return

    if not os.path.exists(p1) or not os.path.exists(p2):
        return

    # Execute a go program
    _ = subprocess.run(["go", "run", ".", p1, p2])

with open("/home/dominykas/results_similar_images_pretty.json", "r") as f:
    data: list[dict] = json.load(f)
    for collection in data:
        for i in range(len(collection)):
            for j in range(i + 1, len(collection)):
                compare_pair(collection[i], collection[j])
