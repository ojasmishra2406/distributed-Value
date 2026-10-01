import json
import os

class FeatureSchemaRegistry:
    def __init__(self, registry_path="schema_registry.json"):
        self.registry_path = registry_path
        self.schemas = self._load()

    def _load(self):
        if os.path.exists(self.registry_path):
            with open(self.registry_path, 'r') as f:
                return json.load(f)
        return {}

    def register_schema(self, version: str, features: list):
        self.schemas[version] = {"features": features}
        with open(self.registry_path, 'w') as f:
            json.dump(self.schemas, f, indent=2)

    def get_schema(self, version: str):
        return self.schemas.get(version)
