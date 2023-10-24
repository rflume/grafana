import { css } from '@emotion/css';
import React, { useEffect, useState } from 'react';

import { GrafanaTheme2, QueryEditorProps, TimeRange } from '@grafana/data';
import { getBackendSrv, getPluginLinkExtensions } from '@grafana/runtime';
import { LinkButton, useStyles2 } from '@grafana/ui';

import { PyroscopeDataSource } from '../datasource';
import { PyroscopeDataSourceOptions, Query } from '../types';

const EXTENSION_POINT_ID = 'plugins/grafana-pyroscope-datasource/query-links';
const DESCRIPTION_INDICATING_CONFIGURATION_NOT_READY = 'configuration-not-ready-yet';

/** A subset of the datasource settings that are relevant for this integration */
type PyroscopeDatasourceSettings = {
  uid: string;
  url: string;
  basicAuthUser: string;
};

/** The context object that will be shared with the link extension's configure function */
type ExtensionQueryLinksContext = {
  datasourceUid: string;
  query: Query;
  range?: TimeRange | undefined;
  datasourceSettings?: PyroscopeDatasourceSettings;
};

/* Global promises to fetch pyroscope datasource settings by uid as encountered */
const pyroscopeDatasourceSettingsByUid: Record<string, Promise<PyroscopeDatasourceSettings>> = {};

/* Reset promises for testing purposes */
export function resetPyroscopeQueryLinkExtensionsFetches() {
  Object.keys(pyroscopeDatasourceSettingsByUid).forEach((key) => delete pyroscopeDatasourceSettingsByUid[key]);
}

/** A subset of the `PyroscopeDataSource` `QueryEditorProps` */
export type Props = Pick<
  QueryEditorProps<PyroscopeDataSource, Query, PyroscopeDataSourceOptions>,
  'datasource' | 'query' | 'range'
>;

export function PyroscopeQueryLinkExtensions(props: Props) {
  const {
    datasource: { uid: datasourceUid },
    query,
    range,
  } = props;

  const [datasourceSettings, setDatasourceSettings] = useState<PyroscopeDatasourceSettings>();
  const [waitingOnExtensionConfigure, setWaitingOnExtensionConfigure] = useState(false);

  const context: ExtensionQueryLinksContext = {
    datasourceUid,
    query,
    range,
    datasourceSettings,
  };

  const { extensions } = getPluginLinkExtensions({
    extensionPointId: EXTENSION_POINT_ID,
    context,
  });

  if (!waitingOnExtensionConfigure) {
    const delayedExtension = extensions.find(
      (extension) => extension.description === DESCRIPTION_INDICATING_CONFIGURATION_NOT_READY
    );

    if (delayedExtension) {
      // Declare that we are waiting on the extension configuration
      setWaitingOnExtensionConfigure(true);

      // Wait a second, and then declare that we are no longer waiting.
      // This trigger another `configure` call to each extension.
      setTimeout(() => setWaitingOnExtensionConfigure(false), 1000);
    }
  }

  const styles = useStyles2(getStyles);

  useEffect(() => {
    let datasourceSettings = pyroscopeDatasourceSettingsByUid[datasourceUid];

    if (datasourceSettings == null) {
      // This explicit fetch of the datasource by its id ensures that we obtain its authentication settings
      datasourceSettings = getBackendSrv().get<PyroscopeDatasourceSettings>(`/api/datasources/uid/${datasourceUid}`);
      pyroscopeDatasourceSettingsByUid[datasourceUid] = datasourceSettings;
    }

    datasourceSettings.then(setDatasourceSettings, () => setDatasourceSettings(undefined));
  }, [datasourceUid]);

  if (extensions.length === 0) {
    return null;
  }

  const configuredExtensions = extensions.filter(
    (extension) => extension.description !== DESCRIPTION_INDICATING_CONFIGURATION_NOT_READY
  );

  return (
    <>
      {configuredExtensions.map((extension) => (
        <LinkButton
          className={styles.linkButton}
          key={`${extension.id}`}
          variant="secondary"
          icon={extension.icon || 'external-link-alt'}
          tooltip={extension.description}
          target="_blank"
          href={extension.path}
          onClick={extension.onClick}
        >
          {extension.title}
        </LinkButton>
      ))}
    </>
  );
}

function getStyles(theme: GrafanaTheme2) {
  return {
    linkButton: css({
      marginLeft: theme.spacing(1),
    }),
  };
}
