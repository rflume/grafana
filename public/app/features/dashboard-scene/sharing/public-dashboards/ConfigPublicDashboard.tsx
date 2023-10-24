import { css } from '@emotion/css';
import React from 'react';
import { useAsync } from 'react-use';

import { GrafanaTheme2 } from '@grafana/data';
import { SceneComponentProps, sceneGraph } from '@grafana/scenes';
import { useStyles2 } from '@grafana/ui';
import { contextSrv } from 'app/core/core';
import ConfigPublicDashboardComponent from 'app/features/dashboard/components/ShareModal/SharePublicDashboard/ConfigPublicDashboard/ConfigPublicDashboard';
import { AccessControlAction } from 'app/types';

import { ShareModal } from '../ShareModal';

import { ConfirmModal } from './ConfirmModal';
import { getUnsupportedDashboardDatasources, panelTypes } from './CreatePublicDashboard';
import { SharePublicDashboardTab } from './SharePublicDashboardTab';

export function ConfigPublicDashboard({ model }: SceneComponentProps<SharePublicDashboardTab>) {
  const styles = useStyles2(getStyles);

  const hasWritePermissions = contextSrv.hasPermission(AccessControlAction.DashboardsPublicWrite);
  const { dashboardRef, publicDashboard, isGetLoading, isUpdateLoading } = model.useState();
  const dashboard = dashboardRef.resolve();
  const { title: dashboardTitle, isDirty } = dashboard.useState();

  const hasTemplateVariables = (dashboard.state.$variables?.state.variables.length ?? 0) > 0;
  const { value: unsupportedDataSources } = useAsync(async () => {
    const types = panelTypes(dashboard);
    return getUnsupportedDashboardDatasources(types);
  }, []);

  const isDataLoading = isUpdateLoading || isGetLoading;
  const timeRangeState = sceneGraph.getTimeRange(model);
  const timeRange = timeRangeState.useState();

  return (
    <ConfigPublicDashboardComponent
      publicDashboard={publicDashboard}
      unsupportedDatasources={unsupportedDataSources}
      isLoading={isDataLoading}
      onUpdate={model.onUpdate}
      onRevoke={() => {
        dashboard.showModal(
          new ConfirmModal({
            isOpen: true,
            title: 'Revoke public URL',
            icon: 'trash-alt',
            confirmText: 'Revoke public URL',
            body: (
              <p className={styles.description}>
                {dashboardTitle
                  ? 'Are you sure you want to revoke this URL? The dashboard will no longer be public.'
                  : 'Orphaned public dashboard will no longer be public.'}
              </p>
            ),
            onDismiss: () => {
              dashboard.showModal(new ShareModal({ dashboardRef, activeTab: 'Public Dashboard' }));
            },
            onConfirm: () => {
              model.onDelete();
              dashboard.closeModal();
            },
          })
        );
      }}
      timeRange={timeRange.value}
      showSaveChangesAlert={hasWritePermissions && isDirty}
      hasTemplateVariables={hasTemplateVariables}
    />
  );
}

const getStyles = (theme: GrafanaTheme2) => ({
  description: css({
    fontSize: theme.typography.body.fontSize,
  }),
});
