#!/bin/bash

namespace="kolony"
helm upgrade kolony -f values.yaml -n ${namespace} --debug --wait .
