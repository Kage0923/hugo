// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hugolib

import (
	"sync"

	"github.com/gohugoio/hugo/resources/page"
)

type pagePaginator struct {
	paginatorInit sync.Once
	current       *page.Pager

	source *pageState
}

func (p *pagePaginator) Paginate(seq interface{}, options ...interface{}) (*page.Pager, error) {
	var initErr error
	p.paginatorInit.Do(func() {
		pagerSize, err := page.ResolvePagerSize(p.source.s.Cfg, options...)
		if err != nil {
			initErr = err
			return
		}

		pd := p.source.targetPathDescriptor
		pd.Type = p.source.outputFormat()
		paginator, err := page.Paginate(pd, seq, pagerSize)
		if err != nil {
			initErr = err
			return
		}

		p.current = paginator.Pagers()[0]

	})

	if initErr != nil {
		return nil, initErr
	}

	return p.current, nil
}

func (p *pagePaginator) Paginator(options ...interface{}) (*page.Pager, error) {
	var initErr error
	p.paginatorInit.Do(func() {
		pagerSize, err := page.ResolvePagerSize(p.source.s.Cfg, options...)
		if err != nil {
			initErr = err
			return
		}

		pd := p.source.targetPathDescriptor
		pd.Type = p.source.outputFormat()
		paginator, err := page.Paginate(pd, p.source.Pages(), pagerSize)
		if err != nil {
			initErr = err
			return
		}

		p.current = paginator.Pagers()[0]

	})

	if initErr != nil {
		return nil, initErr
	}

	return p.current, nil
}
