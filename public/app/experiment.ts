// create URLSearchParams object from window.location
const params = new URLSearchParams(window.location.search);

export const shadowMode = params.get('shadowMode');
