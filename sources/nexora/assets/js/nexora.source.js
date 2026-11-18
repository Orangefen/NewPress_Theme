/* Native links and forms remain usable when enhancements are unavailable. */
import { initNavigation } from './modules/navigation.js';
import { initNavigationState } from './modules/navigation-state.js';
import { initSearch } from './modules/search.js';
import { initCarousel } from './modules/carousel.js';
import { initReading, initCommentReplies } from './modules/reading.js';

document.documentElement.classList.add('js');
const navigation = initNavigation(document, window);
initNavigationState(document, window);
initSearch(document, window, navigation);
initCarousel(document);
initReading();
initCommentReplies();
