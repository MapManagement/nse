#!/bin/bash

createdb nse_dev
# necessary to prevent error
# FATAL:  role "root" does not exist
# FATAL:  database "root" does not exist
createuser root
createdb root
