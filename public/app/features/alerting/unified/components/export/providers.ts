interface ExportProvider<TFormat> {
  name: string;
  exportFormat: TFormat;
  formatter?: (raw: string) => string;
}

const JsonExportProvider: ExportProvider<'json'> = {
  name: 'JSON',
  exportFormat: 'json',
  formatter: (raw: string) => {
    try {
      return JSON.stringify(JSON.parse(raw), null, 4);
    } catch (e) {
      return raw;
    }
  },
};

const YamlExportProvider: ExportProvider<'yaml'> = {
  name: 'YAML',
  exportFormat: 'yaml',
};

const HclExportProvider: ExportProvider<'hcl'> = {
  name: 'HCL',
  exportFormat: 'hcl',
};

export const grafanaExportProviders = {
  [JsonExportProvider.exportFormat]: JsonExportProvider,
  [YamlExportProvider.exportFormat]: YamlExportProvider,
  [HclExportProvider.exportFormat]: HclExportProvider,
} as const;

export type ExportFormats = keyof typeof grafanaExportProviders;