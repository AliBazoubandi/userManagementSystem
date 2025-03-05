import os
import yaml
import base64

# Generate a random 32-byte key
jwt_key = base64.urlsafe_b64encode(os.urandom(32)).decode('utf-8')

with open('../config/config.yaml', 'r') as file:
    config = yaml.safe_load(file)

config['jwt']['key'] = jwt_key

with open('../config/config.yaml', 'w') as file:
    yaml.safe_dump(config, file)

print(f"Generated JWT key: {jwt_key}")