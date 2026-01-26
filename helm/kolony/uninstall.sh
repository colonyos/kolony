#!/bin/bash

namespace="kolony"

helm uninstall kolony -n ${namespace}
