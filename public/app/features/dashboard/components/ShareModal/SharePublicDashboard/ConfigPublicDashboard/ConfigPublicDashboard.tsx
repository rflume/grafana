import { css } from '@emotion/css';
import React from 'react';
import { useForm } from 'react-hook-form';

import { GrafanaTheme2, TimeRange } from '@grafana/data/src';
import { selectors as e2eSelectors } from '@grafana/e2e-selectors/src';
import { config, featureEnabled } from '@grafana/runtime/src';
import { Button, ClipboardButton, Field, HorizontalGroup, Input, Label, Switch, useStyles2 } from '@grafana/ui/src';
import { Layout } from '@grafana/ui/src/components/Layout/Layout';

import { contextSrv } from '../../../../../../core/services/context_srv';
import { AccessControlAction } from '../../../../../../types';
import { useIsDesktop } from '../../../../utils/screen';
import { trackDashboardSharingActionPerType } from '../../analytics';
import { shareDashboardType } from '../../utils';
import { NoUpsertPermissionsAlert } from '../ModalAlerts/NoUpsertPermissionsAlert';
import { SaveDashboardChangesAlert } from '../ModalAlerts/SaveDashboardChangesAlert';
import { UnsupportedDataSourcesAlert } from '../ModalAlerts/UnsupportedDataSourcesAlert';
import { UnsupportedTemplateVariablesAlert } from '../ModalAlerts/UnsupportedTemplateVariablesAlert';
import { generatePublicDashboardUrl, PublicDashboard } from '../SharePublicDashboardUtils';

import { Configuration } from './Configuration';
import { EmailSharingConfiguration } from './EmailSharingConfiguration';
import { SettingsBar } from './SettingsBar';
import { SettingsSummary } from './SettingsSummary';

const selectors = e2eSelectors.pages.ShareDashboardModal.PublicDashboard;

export interface ConfigPublicDashboardForm {
  isAnnotationsEnabled: boolean;
  isTimeSelectionEnabled: boolean;
  isPaused: boolean;
}

interface Props {
  unsupportedDatasources?: string[];
  showSaveChangesAlert?: boolean;
  publicDashboard?: PublicDashboard;
  isLoading?: boolean;
  hasTemplateVariables?: boolean;
  timeRange: TimeRange;
  onUpdate: (p: PublicDashboard) => void;
  onRevoke: () => void;
}

const ConfigPublicDashboard = ({
  onRevoke,
  timeRange,
  hasTemplateVariables = false,
  showSaveChangesAlert = false,
  onUpdate,
  isLoading = false,
  unsupportedDatasources = [],
  publicDashboard,
}: Props) => {
  const styles = useStyles2(getStyles);
  const isDesktop = useIsDesktop();

  const hasWritePermissions = contextSrv.hasPermission(AccessControlAction.DashboardsPublicWrite);
  const hasEmailSharingEnabled =
    !!config.featureToggles.publicDashboardsEmailSharing && featureEnabled('publicDashboardsEmailSharing');

  const disableInputs = !hasWritePermissions || isLoading;

  const { handleSubmit, setValue, register } = useForm<ConfigPublicDashboardForm>({
    defaultValues: {
      isAnnotationsEnabled: publicDashboard?.annotationsEnabled,
      isTimeSelectionEnabled: publicDashboard?.timeSelectionEnabled,
      isPaused: !publicDashboard?.isEnabled,
    },
  });

  const onPublicDashboardUpdate = async (values: ConfigPublicDashboardForm) => {
    const { isAnnotationsEnabled, isTimeSelectionEnabled, isPaused } = values;

    onUpdate({
      ...publicDashboard!,
      annotationsEnabled: isAnnotationsEnabled,
      timeSelectionEnabled: isTimeSelectionEnabled,
      isEnabled: !isPaused,
    });
  };

  const onChange = async (name: keyof ConfigPublicDashboardForm, value: boolean) => {
    setValue(name, value);
    await handleSubmit((data) => onPublicDashboardUpdate(data))();
  };

  function onCopyURL() {
    trackDashboardSharingActionPerType('copy_public_url', shareDashboardType.publicDashboard);
  }

  return (
    <div className={styles.configContainer}>
      {showSaveChangesAlert && <SaveDashboardChangesAlert />}
      {!hasWritePermissions && <NoUpsertPermissionsAlert mode="edit" />}
      {hasTemplateVariables && <UnsupportedTemplateVariablesAlert />}
      {unsupportedDatasources.length > 0 && (
        <UnsupportedDataSourcesAlert unsupportedDataSources={unsupportedDatasources.join(', ')} />
      )}

      {hasEmailSharingEnabled && <EmailSharingConfiguration />}

      <Field label="Dashboard URL" className={styles.fieldSpace}>
        <Input
          value={generatePublicDashboardUrl(publicDashboard!.accessToken!)}
          readOnly
          disabled={!publicDashboard?.isEnabled}
          data-testid={selectors.CopyUrlInput}
          addonAfter={
            <ClipboardButton
              data-testid={selectors.CopyUrlButton}
              variant="primary"
              disabled={!publicDashboard?.isEnabled}
              getText={() => generatePublicDashboardUrl(publicDashboard!.accessToken!)}
              onClipboardCopy={onCopyURL}
            >
              Copy
            </ClipboardButton>
          }
        />
      </Field>

      <Field className={styles.fieldSpace}>
        <Layout>
          <Switch
            {...register('isPaused')}
            disabled={disableInputs}
            onChange={(e) => {
              trackDashboardSharingActionPerType(
                e.currentTarget.checked ? 'disable_sharing' : 'enable_sharing',
                shareDashboardType.publicDashboard
              );
              onChange('isPaused', e.currentTarget.checked);
            }}
            data-testid={selectors.PauseSwitch}
          />
          <Label
            className={css`
              margin-bottom: 0;
            `}
          >
            Pause sharing dashboard
          </Label>
        </Layout>
      </Field>

      <Field className={styles.fieldSpace}>
        <SettingsBar
          title="Settings"
          headerElement={({ className }) => (
            <SettingsSummary
              className={className}
              isDataLoading={isLoading}
              timeRange={timeRange}
              timeSelectionEnabled={publicDashboard?.timeSelectionEnabled}
              annotationsEnabled={publicDashboard?.annotationsEnabled}
            />
          )}
          data-testid={selectors.SettingsDropdown}
        >
          <Configuration disabled={disableInputs} onChange={onChange} register={register} timeRange={timeRange} />
        </SettingsBar>
      </Field>

      <Layout
        orientation={isDesktop ? 0 : 1}
        justify={isDesktop ? 'flex-end' : 'flex-start'}
        align={isDesktop ? 'center' : 'normal'}
      >
        <HorizontalGroup justify="flex-end">
          <Button
            aria-label="Revoke public URL"
            title="Revoke public URL"
            onClick={onRevoke}
            type="button"
            disabled={disableInputs}
            data-testid={selectors.DeleteButton}
            variant="destructive"
            fill="outline"
          >
            Revoke public URL
          </Button>
        </HorizontalGroup>
      </Layout>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
  configContainer: css`
    label: config container;
    display: flex;
    flex-direction: column;
    flex-wrap: wrap;
    gap: ${theme.spacing(3)};
  `,
  fieldSpace: css`
    label: field space;
    width: 100%;
    margin-bottom: 0;
  `,
  timeRange: css({
    display: 'inline-block',
  }),
});

export default ConfigPublicDashboard;
