# SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
# SPDX-License-Identifier: Apache-2.0

import os.path
from distutils.command.upload import upload as upload_orig

from setuptools import find_namespace_packages, setup


class Upload(upload_orig):
    def _get_rc_file(self):
        return os.path.join(".", ".pypirc")

    package_data = ({"ml": ["dir/dummy/*"]},)


setup(
    name="ml-gen_model",
    description="Super complex package",
    long_description="Super complex package + 1",
    test_suite="tests",
    url="https://github.com/agntcy/dir",
    packages=find_namespace_packages(),
    zip_safe=False,
    version="0.0.3",
    cmdclass={"upload": Upload},
)
