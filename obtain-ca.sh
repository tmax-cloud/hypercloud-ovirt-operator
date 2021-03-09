#!/bin/bash
openssl s_client -connect master.tmax.dom:443 -showcerts < /dev/null