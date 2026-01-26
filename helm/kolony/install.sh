#!/bin/bash

namespace="kolony"
helm install kolony -f values.yaml -n ${namespace} .
