import React, { useState } from 'react';
import { css } from '@emotion/css';
import { Button, CodeEditor, Modal, useTheme2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { getBackendSrv, config } from '@grafana/runtime';

// export enum CrawlerMode {
// 	Thumbs = 'thumbs',
// 	Analytics = 'analytics', // Enterprise only
// 	Migrate = 'migrate',
// }

export const CrawlerStartButton = () => {
  const styles = getStyles(useTheme2());
  const [open, setOpen] = useState(false);
  const [body, setBody] = useState({
    mode: 'thumbs',
    user: 'admin',
    password: 'admin',
    theme: config.theme2.isLight ? 'light' : 'dark',
    limit: 1000,
    concurrency: 1,
  });
  const onDismiss = () => setOpen(false);
  const doStart = () => {
    getBackendSrv()
      .post('/api/admin/crawler/start', body)
      .then((v) => {
        console.log('GOT', v);
        onDismiss();
      });
  };

  return (
    <>
      <Modal title={'Start crawler'} isOpen={open} onDismiss={onDismiss}>
        <div className={styles.wrap}>
          For now, we must enter credentials manually :(
          <CodeEditor
            height={200}
            value={JSON.stringify(body, null, 2) ?? ''}
            showLineNumbers={false}
            readOnly={false}
            language="json"
            showMiniMap={false}
            onBlur={(text: string) => {
              setBody(JSON.parse(text)); // force JSON?
            }}
          />
        </div>
        <Modal.ButtonRow>
          <Button onClick={doStart}>Start</Button>
          <Button variant="secondary" onClick={onDismiss}>
            Cancel
          </Button>
        </Modal.ButtonRow>
      </Modal>

      <Button onClick={() => setOpen(true)} variant="primary">
        Start
      </Button>
    </>
  );
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    wrap: css`
      border: 2px solid #111;
    `,
  };
};
